package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/proxy"
	"github.com/steig/tube/internal/service"
)

// NewAddCmd creates the add command
func NewAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> <port>",
		Short: "Add a project to tube",
		Long: `Add a new project to tube.

Examples:
  tube add myapp 3000      # Add React app on port 3000
  tube add api 8080        # Add API server on port 8080
  tube add storybook 6006  # Add Storybook on port 6006`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			portStr := args[1]

			// Get config path from flag
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Parse port
			port, err := ParsePort(portStr)
			if err != nil {
				return err
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

			// Add the project
			if err := AddProject(cfg, configPath, pm, ngx, dms, name, port); err != nil {
				return err
			}

			cmd.Printf("✓ Added project '%s' on port %d\n", name, port)
			cmd.Printf("  Local:  http://%s.%s\n", name, cfg.Proxy.LocalDomain)
			cmd.Printf("  Public: https://%s%s.%s (when tunnel enabled)\n", cfg.TunnelPrefix, name, cfg.Domain)
			return nil
		},
	}
}
