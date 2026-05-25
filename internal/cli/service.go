package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewStartCmd creates the start command
func NewStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start tube services (nginx, dnsmasq)",
		Long:  `Start the tube proxy services.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := loadStack(cmd)
			if err != nil {
				return err
			}

			cmd.Println("Generating configurations...")
			if err := st.ngx.WriteConfig(); err != nil {
				return fmt.Errorf("failed to generate nginx config: %w", err)
			}
			if err := st.dms.WriteConfig(); err != nil {
				return fmt.Errorf("failed to generate dnsmasq config: %w", err)
			}

			cmd.Println("Starting services...")
			if err := st.ngx.Start(); err != nil {
				return fmt.Errorf("failed to start nginx: %w", err)
			}
			if err := st.dms.Start(); err != nil {
				_ = st.ngx.Stop() // rollback
				return fmt.Errorf("failed to start dnsmasq: %w", err)
			}

			cmd.Println("✓ Services started successfully")
			cmd.Printf("  nginx listening on :%d, :%d\n", st.cfg.Nginx.HTTPPort, st.cfg.Nginx.HTTPSPort)
			cmd.Printf("  dnsmasq listening on :%d\n", st.cfg.Dnsmasq.Port)
			cmd.Printf("  Projects available at: *%s\n", st.cfg.Proxy.LocalDomain)
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
			st, err := loadStack(cmd)
			if err != nil {
				return err
			}

			cmd.Println("Stopping services...")
			if err := st.pm.StopAll(); err != nil {
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
			st, err := loadStack(cmd)
			if err != nil {
				return err
			}

			cmd.Println("Restarting services...")
			if err := st.pm.RestartAll(); err != nil {
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
			st, err := loadStack(cmd)
			if err != nil {
				return err
			}

			cmd.Println("Service Status:")
			for _, svc := range []string{"nginx", "dnsmasq"} {
				status, err := st.pm.Status(svc)
				if err != nil {
					status = "unknown"
				}
				indicator := "○"
				if running, _ := st.pm.IsRunning(svc); running {
					indicator = "●"
				}
				cmd.Printf("  %-10s %s %s\n", svc+":", indicator, status)
			}
			return nil
		},
	}
}
