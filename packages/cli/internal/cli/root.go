package cli

import (
	"fmt"
	"os"

	"github.com/dxta-dev/clankers/internal/paths"
	"github.com/spf13/cobra"
)

var (
	// Version info (set at build time)
	Version   = "dev"
	BuildTime = "unknown"

	// Global flags
	configPath string
)

// RootCmd is the base command for the CLI
func RootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "clankers",
		Short: "Clankers - AI session tracking and sync",
		Long: `Clankers tracks your AI coding sessions and syncs them across devices.

The CLI provides commands to manage the local daemon, query your session data,
and configure sync settings.

Usage:
  clankers daemon          Run the background daemon
  clankers config          Manage configuration
  clankers query           Query session data
  clankers sync            Sync operations
`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// No subcommand specified - show help and error
			cmd.Help()
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "Error: No subcommand specified. Use 'clankers daemon' to start the daemon.")
			return fmt.Errorf("no subcommand specified")
		},
	}

	// Global flags
	root.PersistentFlags().StringVar(&configPath, "config", "", fmt.Sprintf("config file path (default: %s)", paths.GetConfigPath()))
	root.PersistentFlags().String("profile", "", "active profile (env: CLANKERS_PROFILE)")

	// Add subcommands
	root.AddCommand(daemonCmd())
	root.AddCommand(configCmd())
	// TODO: Add sync command in Phase 4
	root.AddCommand(queryCmd())
	// root.AddCommand(syncCmd())

	return root
}

// Execute runs the root command
func Execute() error {
	return RootCmd().Execute()
}
