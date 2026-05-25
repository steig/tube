package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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

			st, err := loadStack(cmd)
			if err != nil {
				return err
			}

			if !ProjectExists(st.cfg, name) {
				return fmt.Errorf("project %q not found", name)
			}

			if err := RemoveProject(st.cfg, st.configPath, st.pm, st.ngx, st.dms, name); err != nil {
				return err
			}

			cmd.Printf("✓ Removed project '%s'\n", name)
			return nil
		},
	}
}
