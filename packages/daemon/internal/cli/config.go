package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dxta-dev/clankers-daemon/internal/config"
	"github.com/spf13/cobra"
)

// configCmd returns the config command group
func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "Manage clankers configuration including profiles, endpoints, and sync settings.",
	}

	cmd.AddCommand(configSetCmd())
	cmd.AddCommand(configGetCmd())
	cmd.AddCommand(configListCmd())
	cmd.AddCommand(configProfilesCmd())

	return cmd
}

// configSetCmd returns the 'config set' command
func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value for the active profile.

Available keys:
  endpoint       - Sync endpoint URL
  sync_enabled   - Enable/disable sync (true/false)
  sync_interval  - Sync interval in seconds
  auth           - Authentication mode

Examples:
  clankers config set endpoint https://my-server.com
  clankers config set sync_enabled true
  clankers config set sync_interval 60`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if err := cfg.SetProfileValue(key, value); err != nil {
				return err
			}

			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Set %s = %s (profile: %s)\n", key, value, cfg.ActiveProfile)
			return nil
		},
	}
}

// configGetCmd returns the 'config get' command
func configGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a configuration value from the active profile.

Examples:
  clankers config get endpoint
  clankers config get sync_enabled`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			value, err := cfg.GetProfileValue(key)
			if err != nil {
				return err
			}

			fmt.Println(value)
			return nil
		},
	}
}

// configListCmd returns the 'config list' command
func configListCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration",
		Long:  "List all configuration for the active profile.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			profile := cfg.GetActiveProfile()

			switch format {
			case "json":
				output := map[string]interface{}{
					"profile":       cfg.ActiveProfile,
					"endpoint":      profile.Endpoint,
					"sync_enabled":  profile.SyncEnabled,
					"sync_interval": profile.SyncInterval,
					"auth":          profile.AuthMode,
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(output)

			default:
				fmt.Printf("Profile: %s\n", cfg.ActiveProfile)
				fmt.Printf("  endpoint:       %s\n", profile.Endpoint)
				fmt.Printf("  sync_enabled:   %t\n", profile.SyncEnabled)
				fmt.Printf("  sync_interval:  %d seconds\n", profile.SyncInterval)
				fmt.Printf("  auth:           %s\n", profile.AuthMode)
				return nil
			}
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text, json")

	return cmd
}

// configProfilesCmd returns the 'config profiles' command group
func configProfilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage profiles",
		Long:  "List and switch between configuration profiles.",
	}

	cmd.AddCommand(profilesListCmd())
	cmd.AddCommand(profilesUseCmd())

	return cmd
}

// profilesListCmd returns the 'config profiles list' command
func profilesListCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available profiles",
		Long:  "List all available configuration profiles and indicate the active one.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			switch format {
			case "json":
				type profileInfo struct {
					Name   string `json:"name"`
					Active bool   `json:"active"`
				}
				var profiles []profileInfo
				for name := range cfg.Profiles {
					profiles = append(profiles, profileInfo{
						Name:   name,
						Active: name == cfg.ActiveProfile,
					})
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(profiles)

			default:
				for name := range cfg.Profiles {
					if name == cfg.ActiveProfile {
						fmt.Printf("* %s (active)\n", name)
					} else {
						fmt.Printf("  %s\n", name)
					}
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text, json")

	return cmd
}

// profilesUseCmd returns the 'config profiles use' command
func profilesUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Switch to a different profile",
		Long:  "Switch the active configuration profile.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if err := cfg.SetActiveProfile(name); err != nil {
				return err
			}

			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Switched to profile: %s\n", name)
			return nil
		},
	}
}
