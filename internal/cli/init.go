package cli

import (
	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/config"
)

// NewInitCmd creates the init command
func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize tube configuration",
		Long: `Initialize tube configuration with interactive prompts for:
- Domain name (e.g., example.com)
- Tunnel prefix (e.g., dev-)
- Local development TLD (default: .test)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := resolveConfigPath(cmd)
			_, err := config.InteractiveInit(configPath)
			return err
		},
	}
}
