package cli

import (
	"path/filepath"
	"os"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/config"
)

// NewInitCmd creates the init command
func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize tube configuration",
		Long: `Initialize tube configuration with interactive prompts for:
- Domain name for public tunnels (e.g., example.com)
- Tunnel prefix for subdomains (e.g., dev-)
- Local development TLD (default: .test)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := filepath.Join(os.Getenv("HOME"), ".tube", "config.yaml")
			_, err := config.InteractiveInit(configPath)
			return err
		},
	}
}
