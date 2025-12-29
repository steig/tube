package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/proxy"
	"github.com/steig/tube/internal/service"
)

// NewRemoveCmd creates the remove command
func NewRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a project from tube",
		Long: `Remove a project from tube configuration.

Example:
  tube remove myapp`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Get config path from flag
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Check if project exists
			if !ProjectExists(cfg, name) {
				return fmt.Errorf("project %q not found", name)
			}

			// Create ProcessManager
			pm, err := service.NewProcessManager(cfg.Directories.PIDs)
			if err != nil {
				return fmt.Errorf("failed to create process manager: %w", err)
			}

			// Create NginxManager
			ngx, err := proxy.NewNginxManager(cfg, pm)
			if err != nil {
				return fmt.Errorf("failed to create nginx manager: %w", err)
			}

			// Create DnsmasqManager
			dms, err := proxy.NewDnsmasqManager(cfg, pm)
			if err != nil {
				return fmt.Errorf("failed to create dnsmasq manager: %w", err)
			}

			// Remove the project
			if err := RemoveProject(cfg, configPath, pm, ngx, dms, name); err != nil {
				return err
			}

			cmd.Printf("✓ Removed project '%s'\n", name)
			return nil
		},
	}
}
