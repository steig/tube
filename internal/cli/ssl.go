package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/ssl"
)

// NewSSLCmd creates the ssl command group
func NewSSLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssl",
		Short: "Manage SSL certificates",
		Long:  `Manage SSL certificates for HTTPS support using mkcert.`,
	}

	cmd.AddCommand(
		NewSSLStatusCmd(),
		NewSSLInstallCmd(),
		NewSSLGenerateCmd(),
		NewSSLEnableCmd(),
		NewSSLDisableCmd(),
	)

	return cmd
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// NewSSLStatusCmd creates the ssl status command
func NewSSLStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show SSL configuration and certificate status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadCfg(cmd)
			if err != nil {
				return err
			}

			cm, err := ssl.NewCertManager(cfg)
			if err != nil {
				cmd.Println("SSL Status:")
				cmd.Println("  Enabled:       ", yesNo(cfg.SSL.Enabled))
				cmd.Println("  mkcert:         not installed")
				cmd.Println()
				cmd.Println("Install mkcert with: brew install mkcert")
				return nil
			}

			s := cm.GetStatus()
			cmd.Println("SSL Status:")
			cmd.Println("  Enabled:       ", yesNo(s.Enabled))
			cmd.Println("  CA Installed:  ", yesNo(s.CAInstalled))
			cmd.Println("  Cert Exists:   ", yesNo(s.CertExists))
			cmd.Println("  Local Domain:  ", s.LocalDomain)
			cmd.Println()
			cmd.Println("Paths:")
			cmd.Println("  mkcert:        ", s.MkcertPath)
			cmd.Println("  Certificate:   ", s.CertFile)
			cmd.Println("  Private Key:   ", s.KeyFile)

			if !s.CAInstalled {
				cmd.Println()
				cmd.Println("Run 'tube ssl install' to install the CA certificate")
			} else if !s.CertExists {
				cmd.Println()
				cmd.Println("Run 'tube ssl generate' to generate certificates")
			}
			return nil
		},
	}
}

// NewSSLInstallCmd creates the ssl install command
func NewSSLInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install mkcert CA to system trust store",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, configPath, err := loadCfg(cmd)
			if err != nil {
				return err
			}

			cm, err := ssl.NewCertManager(cfg)
			if err != nil {
				return fmt.Errorf("mkcert not available: %w", err)
			}

			if cm.IsCAInstalled() {
				cmd.Println("CA certificate is already installed")
				return nil
			}

			cmd.Println("Installing mkcert CA certificate...")
			if err := cm.InstallCA(); err != nil {
				return fmt.Errorf("failed to install CA: %w", err)
			}

			cfg.SSL.CAInstalled = true
			if err := cfg.Save(configPath); err != nil {
				cmd.Printf("Warning: could not save config: %v\n", err)
			}

			cmd.Println()
			cmd.Println("CA certificate installed successfully!")
			cmd.Println("You can now generate certificates with: tube ssl generate")
			return nil
		},
	}
}

// NewSSLGenerateCmd creates the ssl generate command
func NewSSLGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate SSL certificates",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, configPath, err := loadCfg(cmd)
			if err != nil {
				return err
			}

			cm, err := ssl.NewCertManager(cfg)
			if err != nil {
				return fmt.Errorf("mkcert not available: %w", err)
			}

			if !cm.IsCAInstalled() {
				cmd.Println("CA certificate not installed. Installing now...")
				if err := cm.InstallCA(); err != nil {
					return fmt.Errorf("failed to install CA: %w", err)
				}
				cfg.SSL.CAInstalled = true
			}

			force, _ := cmd.Flags().GetBool("force")
			if cm.CertExists(cfg.Proxy.LocalDomain) && !force {
				cmd.Println("Certificate already exists for", cfg.Proxy.LocalDomain)
				cmd.Println("Use --force to regenerate")
				return nil
			}

			cmd.Printf("Generating wildcard certificate for *%s...\n", cfg.Proxy.LocalDomain)
			certInfo, err := cm.GenerateWildcard(cfg.Proxy.LocalDomain)
			if err != nil {
				return fmt.Errorf("failed to generate certificate: %w", err)
			}

			cfg.SSL.CertFile = certInfo.CertFile
			cfg.SSL.KeyFile = certInfo.KeyFile
			cfg.SSL.Enabled = true
			if err := cfg.Save(configPath); err != nil {
				cmd.Printf("Warning: could not save config: %v\n", err)
			}

			cmd.Println()
			cmd.Println("Certificate generated successfully!")
			cmd.Println("  Certificate:", certInfo.CertFile)
			cmd.Println("  Private Key:", certInfo.KeyFile)
			cmd.Println()
			cmd.Println("Restart services to apply: tube restart")
			return nil
		},
	}

	cmd.Flags().BoolP("force", "f", false, "Force regeneration of certificates")
	return cmd
}

// NewSSLEnableCmd creates the ssl enable command
func NewSSLEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable HTTPS",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, configPath, err := loadCfg(cmd)
			if err != nil {
				return err
			}

			if cfg.SSL.Enabled {
				cmd.Println("SSL is already enabled")
				return nil
			}

			cm, err := ssl.NewCertManager(cfg)
			if err != nil {
				return fmt.Errorf("mkcert not available: %w", err)
			}

			if !cm.CertExists(cfg.Proxy.LocalDomain) {
				cmd.Println("No certificates found. Generating...")
				if !cm.IsCAInstalled() {
					if err := cm.InstallCA(); err != nil {
						return fmt.Errorf("failed to install CA: %w", err)
					}
					cfg.SSL.CAInstalled = true
				}
				certInfo, err := cm.GenerateWildcard(cfg.Proxy.LocalDomain)
				if err != nil {
					return fmt.Errorf("failed to generate certificate: %w", err)
				}
				cfg.SSL.CertFile = certInfo.CertFile
				cfg.SSL.KeyFile = certInfo.KeyFile
			}

			cfg.SSL.Enabled = true
			if err := cfg.Save(configPath); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			cmd.Println("SSL enabled successfully!")
			cmd.Println("Restart services to apply: tube restart")
			return nil
		},
	}
}

// NewSSLDisableCmd creates the ssl disable command
func NewSSLDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable HTTPS",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, configPath, err := loadCfg(cmd)
			if err != nil {
				return err
			}

			if !cfg.SSL.Enabled {
				cmd.Println("SSL is already disabled")
				return nil
			}

			cfg.SSL.Enabled = false
			if err := cfg.Save(configPath); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			cmd.Println("SSL disabled")
			cmd.Println("Restart services to apply: tube restart")
			return nil
		},
	}
}
