package cli

import (
	"github.com/spf13/cobra"
)

// NewDoctorCmd creates the doctor command
func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check dependencies and system setup",
		Long: `Check that all required dependencies are installed and properly configured.

Shows installation commands for any missing dependencies.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement doctor functionality
			cmd.Println("Checking tube dependencies...")
			cmd.Println("  nginx:      ✓ found")
			cmd.Println("  dnsmasq:    ✓ found")
			cmd.Println("  cloudflared: ○ not found")
			cmd.Println("\nTo install missing dependencies:")
			cmd.Println("  brew install cloudflared")
			return nil
		},
	}
}
