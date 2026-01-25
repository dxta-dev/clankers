package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/dxta-dev/clankers-daemon/internal/paths"
	"github.com/dxta-dev/clankers-daemon/internal/rpc"
	"github.com/dxta-dev/clankers-daemon/internal/storage"
	"github.com/sourcegraph/jsonrpc2"
)

func main() {
	var (
		socketPath string
		dataRoot   string
		dbPath     string
		logLevel   string
	)

	flag.StringVar(&socketPath, "socket", "", "socket path (default: data root + dxta-clankers.sock)")
	flag.StringVar(&dataRoot, "data-root", "", "data root directory (overrides CLANKERS_DATA_PATH)")
	flag.StringVar(&dbPath, "db-path", "", "database file path (overrides CLANKERS_DB_PATH)")
	flag.StringVar(&logLevel, "log-level", "info", "log level: debug, info, warn, error")
	flag.Parse()

	if dataRoot != "" {
		os.Setenv("CLANKERS_DATA_PATH", dataRoot)
	}
	if dbPath != "" {
		os.Setenv("CLANKERS_DB_PATH", dbPath)
	}
	if socketPath == "" {
		socketPath = paths.GetSocketPath()
	}

	resolvedDbPath := paths.GetDbPath()
	created, err := storage.EnsureDb(resolvedDbPath)
	if err != nil {
		log.Fatalf("failed to ensure database: %v", err)
	}
	if created {
		log.Printf("created database at %s", resolvedDbPath)
	}

	store, err := storage.Open(resolvedDbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	if runtime.GOOS != "windows" {
		os.Remove(socketPath)
	}

	var listener net.Listener
	if runtime.GOOS == "windows" {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		log.Printf("listening on %s", listener.Addr())
	} else {
		listener, err = net.Listen("unix", socketPath)
		if err != nil {
			log.Fatalf("failed to listen on %s: %v", socketPath, err)
		}
		log.Printf("listening on %s", socketPath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		cancel()
		listener.Close()
	}()

	handler := rpc.NewHandler(store)
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}

		go serveConn(ctx, conn, handler)
	}
}

func serveConn(ctx context.Context, conn net.Conn, handler *rpc.Handler) {
	defer conn.Close()

	stream := jsonrpc2.NewBufferedStream(conn, jsonrpc2.VSCodeObjectCodec{})
	rpcConn := jsonrpc2.NewConn(
		ctx,
		stream,
		jsonrpc2.HandlerWithError(func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
			handler.Handle(ctx, conn, req)
			return nil, nil
		}),
	)

	<-rpcConn.DisconnectNotify()
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nDefault paths:\n")
		fmt.Fprintf(os.Stderr, "  data root: %s\n", paths.GetDataRoot())
		fmt.Fprintf(os.Stderr, "  database:  %s\n", paths.GetDbPath())
		fmt.Fprintf(os.Stderr, "  socket:    %s\n", paths.GetSocketPath())
	}
}
