package cli

import (
	"strings"

	"github.com/spf13/cobra"
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
			port, err := ParsePort(args[1])
			if err != nil {
				return err
			}

			st, err := loadStack(cmd)
			if err != nil {
				return err
			}

			if err := AddProject(st.cfg, st.configPath, st.pm, st.ngx, st.dms, name, port); err != nil {
				return err
			}

			// strip the leading dot off LocalDomain (".test") so the URL is well-formed.
			localTLD := strings.TrimPrefix(st.cfg.Proxy.LocalDomain, ".")
			scheme := "http"
			if st.cfg.SSL.Enabled {
				scheme = "https"
			}
			cmd.Printf("✓ Added project '%s' on port %d\n", name, port)
			cmd.Printf("  Local:  %s://%s.%s\n", scheme, name, localTLD)
			return nil
		},
	}
}
