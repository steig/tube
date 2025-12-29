package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/config"
)

// NewListCmd creates the list command
func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Long: `List all projects managed by tube.

Shows project name, local URL, port, and running status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from flag
			configPath, _ := cmd.Flags().GetString("config")

			// Load configuration
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Get project list
			projects, err := ListProjects(cfg)
			if err != nil {
				return err
			}

			// Check if no projects
			if len(projects) == 0 {
				cmd.Println("No projects configured. Run 'tube add <name> <port>' to get started.")
				return nil
			}

			// Sort by name for consistent output
			sort.Slice(projects, func(i, j int) bool {
				return projects[i].Name < projects[j].Name
			})

			// Print header
			cmd.Printf("%-20s %-8s %-45s %-10s\n", "NAME", "PORT", "URL", "STATUS")
			cmd.Println("─────────────────────────────────────────────────────────────────────────────────")

			// Print projects
			for _, p := range projects {
				status := "●"
				if !p.Running {
					status = "○"
				}

				cmd.Printf("%-20s %-8d %-45s %s\n", p.Name, p.Port, p.LocalURL, status)
			}

			cmd.Println()
			cmd.Println("● = running  ○ = stopped")
			return nil
		},
	}
}
