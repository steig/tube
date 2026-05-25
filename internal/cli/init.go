package cli

import (
	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/config"
)

// NewInitCmd creates the init command
func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize tube configuration",
		Long: `Initialize tube configuration.

Asks for the local development TLD (default: .test), creates the directory
tree under ~/.tube, and runs mkcert to generate HTTPS certs.

Pass --with-tunnel to additionally configure the Cloudflare Tunnel fields
(domain, tunnel_prefix). Tunnel functionality is not yet implemented, so
those prompts are skipped by default.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := resolveConfigPath(cmd)
			withTunnel, _ := cmd.Flags().GetBool("with-tunnel")
			_, err := config.InteractiveInit(configPath, withTunnel)
			return err
		},
	}
	cmd.Flags().Bool("with-tunnel", false, "also prompt for Cloudflare Tunnel fields (planned feature)")
	return cmd
}
