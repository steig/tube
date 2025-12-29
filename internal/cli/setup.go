package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/dns"
)

// NewSetupCmd creates the setup command
func NewSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Setup tube on your system",
		Long: `Setup tube by configuring macOS DNS resolver for .test domains.

This command will:
1. Create /etc/resolver/test to route *.test domains to localhost
2. Flush the DNS cache

Requires sudo privileges.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from flag
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Get domain from config (strip leading dot)
			domain := cfg.Proxy.LocalDomain

			cmd.Printf("Setting up tube for *.%s domains...\n", domain)

			// Create resolver manager
			rm := dns.NewResolverManager(domain, "127.0.0.1")

			// Check current status
			status, _ := rm.Status()
			if status == "not configured" {
				cmd.Println("Configuring DNS resolver (requires sudo)...")

				if err := rm.SetupWithSudo(); err != nil {
					return fmt.Errorf("failed to setup DNS resolver: %w", err)
				}

				cmd.Println("Flushing DNS cache...")
				if err := dns.FlushDNSCache(); err != nil {
					cmd.Printf("Warning: could not flush DNS cache: %v\n", err)
				}

				cmd.Printf("✓ DNS resolver configured for *.%s\n", domain)
			} else {
				cmd.Printf("✓ DNS resolver already configured for *.%s\n", domain)
			}

			cmd.Println("\nSetup complete! You can now:")
			cmd.Println("  1. Add projects: tube add <name> <port>")
			cmd.Println("  2. Start services: tube start")
			cmd.Println("  3. Access your apps at http://<name>.test")

			return nil
		},
	}

	return cmd
}

// NewUninstallCmd creates the uninstall command
func NewUninstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove tube configuration from your system",
		Long: `Remove tube DNS configuration from your system.

This command will:
1. Stop all running services
2. Remove /etc/resolver/test
3. Flush the DNS cache

Requires sudo privileges.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from flag
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			domain := cfg.Proxy.LocalDomain

			cmd.Println("Removing tube configuration...")

			// Create resolver manager
			rm := dns.NewResolverManager(domain, "127.0.0.1")

			// Remove resolver
			cmd.Println("Removing DNS resolver (requires sudo)...")
			if err := rm.RemoveWithSudo(); err != nil {
				cmd.Printf("Warning: could not remove DNS resolver: %v\n", err)
			}

			// Flush DNS cache
			cmd.Println("Flushing DNS cache...")
			if err := dns.FlushDNSCache(); err != nil {
				cmd.Printf("Warning: could not flush DNS cache: %v\n", err)
			}

			cmd.Println("✓ tube configuration removed")

			return nil
		},
	}

	return cmd
}

// NewDNSStatusCmd creates a command to check DNS status
func NewDNSStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dns-status",
		Short: "Check DNS resolver status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from flag
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			domain := cfg.Proxy.LocalDomain
			rm := dns.NewResolverManager(domain, "127.0.0.1")

			status, err := rm.Status()
			if err != nil {
				return fmt.Errorf("failed to check DNS status: %w", err)
			}

			cmd.Printf("DNS Resolver: %s\n", status)

			return nil
		},
	}
}
