package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCmd creates the root command
func NewRootCmd(version, commit, date string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tube",
		Short: "Local development proxy with .test domains and Cloudflare tunnels",
		Long: `Tube is a local development proxy that lets you:
- Access local projects via .test domains (e.g., myapp.test)
- Expose projects publicly via Cloudflare Tunnel
- Use HTTPS with automatic certificate management
- Monitor services via a beautiful web dashboard`,
		Version: fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
	}

	// Add commands
	cmd.AddCommand(
		NewInitCmd(),
		NewSetupCmd(),
		NewUninstallCmd(),
		NewAddCmd(),
		NewRemoveCmd(),
		NewListCmd(),
		NewStartCmd(),
		NewStopCmd(),
		NewRestartCmd(),
		NewStatusCmd(),
		NewDNSStatusCmd(),
		NewConfigCmd(),
		NewLogsCmd(),
		NewDoctorCmd(),
		NewSSLCmd(),
	)

	// Global flags
	cmd.PersistentFlags().StringP("config", "c", "", "path to config file (default: ~/.tube/config.yaml)")

	return cmd
}
