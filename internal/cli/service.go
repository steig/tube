package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/proxy"
	"github.com/steig/tube/internal/service"
)

// NewStartCmd creates the start command
func NewStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start tube services (nginx, dnsmasq)",
		Long:  `Start the tube proxy services.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Create ProcessManager
			pm, err := service.NewProcessManager(cfg.Directories.PIDs)
			if err != nil {
				return fmt.Errorf("failed to create process manager: %w", err)
			}

			// Create managers
			ngx, err := proxy.NewNginxManager(cfg, pm)
			if err != nil {
				return fmt.Errorf("failed to create nginx manager: %w", err)
			}

			dms, err := proxy.NewDnsmasqManager(cfg, pm)
			if err != nil {
				return fmt.Errorf("failed to create dnsmasq manager: %w", err)
			}

			// Write configurations
			cmd.Println("Generating configurations...")
			if err := ngx.WriteConfig(); err != nil {
				return fmt.Errorf("failed to generate nginx config: %w", err)
			}

			if err := dms.WriteConfig(); err != nil {
				return fmt.Errorf("failed to generate dnsmasq config: %w", err)
			}

			// Start services
			cmd.Println("Starting services...")
			if err := ngx.Start(); err != nil {
				return fmt.Errorf("failed to start nginx: %w", err)
			}

			if err := dms.Start(); err != nil {
				_ = ngx.Stop() // Rollback
				return fmt.Errorf("failed to start dnsmasq: %w", err)
			}

			cmd.Println("✓ Services started successfully")
			cmd.Println("  nginx listening on :80, :443")
			cmd.Println("  dnsmasq listening on :53")
			cmd.Printf("  Projects available at: *.%s\n", cfg.Proxy.LocalDomain)
			return nil
		},
	}
}

// NewStopCmd creates the stop command
func NewStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop tube services",
		Long:  `Stop all tube proxy services.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Create ProcessManager
			pm, err := service.NewProcessManager(cfg.Directories.PIDs)
			if err != nil {
				return fmt.Errorf("failed to create process manager: %w", err)
			}

			// Stop services
			cmd.Println("Stopping services...")
			if err := pm.StopAll(); err != nil {
				return fmt.Errorf("failed to stop services: %w", err)
			}

			cmd.Println("✓ Services stopped")
			return nil
		},
	}
}

// NewRestartCmd creates the restart command
func NewRestartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart tube services",
		Long:  `Restart all tube proxy services.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Create ProcessManager
			pm, err := service.NewProcessManager(cfg.Directories.PIDs)
			if err != nil {
				return fmt.Errorf("failed to create process manager: %w", err)
			}

			// Restart
			cmd.Println("Restarting services...")
			if err := pm.RestartAll(); err != nil {
				return fmt.Errorf("failed to restart services: %w", err)
			}

			cmd.Println("✓ Services restarted")
			return nil
		},
	}
}

// NewStatusCmd creates the status command
func NewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show service status",
		Long:  `Show the status of all tube services.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Create ProcessManager
			pm, err := service.NewProcessManager(cfg.Directories.PIDs)
			if err != nil {
				return fmt.Errorf("failed to create process manager: %w", err)
			}

			// Get status
			cmd.Println("Service Status:")

			services := []string{"nginx", "dnsmasq"}
			for _, svc := range services {
				status, err := pm.Status(svc)
				if err != nil {
					status = "unknown"
				}

				indicator := "○"
				if isRunning, _ := pm.IsRunning(svc); isRunning {
					indicator = "●"
				}

				cmd.Printf("  %-10s %s %s\n", svc+":", indicator, status)
			}

			return nil
		},
	}
}
