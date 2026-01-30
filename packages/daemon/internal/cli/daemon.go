package cli

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/dxta-dev/clankers/internal/logging"
	"github.com/dxta-dev/clankers/internal/paths"
	"github.com/dxta-dev/clankers/internal/rpc"
	"github.com/dxta-dev/clankers/internal/storage"
	"github.com/sourcegraph/jsonrpc2"
	"github.com/spf13/cobra"
)

type filteredLogWriter struct {
	w io.Writer
}

func (f *filteredLogWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	if strings.Contains(s, "connection reset by peer") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "use of closed network connection") ||
		strings.Contains(s, "jsonrpc2: protocol error") && strings.Contains(s, "read unix") {
		return len(p), nil
	}
	return f.w.Write(p)
}

func daemonCmd() *cobra.Command {
	var (
		socketPath string
		dataRoot   string
		dbPath     string
		logLevel   string
	)

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run the background daemon",
		Long: `Run the Clankers daemon that listens for plugin connections
and stores session data to the local database.

The daemon listens on a Unix socket (macOS/Linux) or TCP (Windows)
and accepts JSON-RPC requests from editor plugins.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.SetOutput(&filteredLogWriter{w: os.Stderr})

			if dataRoot != "" {
				os.Setenv("CLANKERS_DATA_PATH", dataRoot)
			}
			if dbPath != "" {
				os.Setenv("CLANKERS_DB_PATH", dbPath)
			}
			if socketPath == "" {
				socketPath = paths.GetSocketPath()
			}

			// Initialize structured logger
			logger, err := logging.New(logLevel, paths.GetLogDir())
			if err != nil {
				// Fall back to stderr logging on error
				log.Printf("failed to initialize logger: %v", err)
				log.Printf("falling back to stderr logging only")
			} else {
				defer logger.Close()
				logger.Infof("daemon", "daemon starting with log level %s", logLevel)
			}

			// Start cleanup job for old log files
			cleanupStop := logging.StartCleanupJob(paths.GetLogDir())
			defer close(cleanupStop)

			resolvedDbPath := paths.GetDbPath()
			created, err := storage.EnsureDb(resolvedDbPath)
			if err != nil {
				return fmt.Errorf("failed to ensure database: %w", err)
			}
			if created {
				if logger != nil {
					logger.Infof("daemon", "created database at %s", resolvedDbPath)
				} else {
					log.Printf("created database at %s", resolvedDbPath)
				}
			}

			store, err := storage.Open(resolvedDbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer store.Close()

			if runtime.GOOS != "windows" {
				os.Remove(socketPath)
			}

			var listener net.Listener
			if runtime.GOOS == "windows" {
				listener, err = net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					return fmt.Errorf("failed to listen: %w", err)
				}
				if logger != nil {
					logger.Infof("daemon", "listening on %s", listener.Addr())
				} else {
					log.Printf("listening on %s", listener.Addr())
				}
			} else {
				listener, err = net.Listen("unix", socketPath)
				if err != nil {
					return fmt.Errorf("failed to listen on %s: %w", socketPath, err)
				}
				if logger != nil {
					logger.Infof("daemon", "listening on %s", socketPath)
				} else {
					log.Printf("listening on %s", socketPath)
				}
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigCh
				if logger != nil {
					logger.Infof("daemon", "shutting down...")
				} else {
					log.Println("shutting down...")
				}
				cancel()
				listener.Close()
			}()

			handler := rpc.NewHandler(store, logger)
			for {
				conn, err := listener.Accept()
				if err != nil {
					select {
					case <-ctx.Done():
						return nil
					default:
						if logger != nil {
							logger.Warnf("daemon", "accept error: %v", err)
						} else {
							log.Printf("accept error: %v", err)
						}
						continue
					}
				}

				go serveConn(ctx, conn, handler)
			}
		},
	}

	cmd.Flags().StringVar(
		&socketPath,
		"socket",
		"",
		"socket path (default: data root + dxta-clankers.sock)",
	)
	cmd.Flags().StringVar(&dataRoot, "data-root", "", "data root directory (overrides CLANKERS_DATA_PATH)")
	cmd.Flags().StringVar(&dbPath, "db-path", "", "database file path (overrides CLANKERS_DB_PATH)")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level: debug, info, warn, error")

	return cmd
}

func serveConn(ctx context.Context, conn net.Conn, handler *rpc.Handler) {
	defer conn.Close()

	stream := jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{})
	rpcConn := jsonrpc2.NewConn(
		ctx,
		stream,
		jsonrpc2.HandlerWithError(
			func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
				handler.Handle(ctx, conn, req)
				return nil, nil
			},
		),
	)

	<-rpcConn.DisconnectNotify()
}
