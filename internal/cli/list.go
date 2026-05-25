package cli

import (
	"sort"

	"github.com/spf13/cobra"
)

// NewListCmd creates the list command
func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Long: `List all projects managed by tube.

Shows project name, local URL, port, and running status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := loadCfg(cmd)
			if err != nil {
				return err
			}

			projects, err := ListProjects(cfg)
			if err != nil {
				return err
			}

			if len(projects) == 0 {
				cmd.Println("No projects configured. Run 'tube add <name> <port>' to get started.")
				return nil
			}

			sort.Slice(projects, func(i, j int) bool {
				return projects[i].Name < projects[j].Name
			})

			cmd.Printf("%-20s %-8s %-45s %-10s\n", "NAME", "PORT", "URL", "STATUS")
			cmd.Println("─────────────────────────────────────────────────────────────────────────────────")

			for _, p := range projects {
				status := "○"
				if p.Running {
					status = "●"
				}
				cmd.Printf("%-20s %-8d %-45s %s\n", p.Name, p.Port, p.LocalURL, status)
			}

			cmd.Println()
			cmd.Println("● = running  ○ = stopped")
			return nil
		},
	}
}
