package cli

import (
	"github.com/spf13/cobra"
)

// NewLogsCmd creates the logs command
func NewLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [type]",
		Short: "View tube logs",
		Long: `View tube logs.

Types: access, error, tunnel, health

Examples:
  tube logs           # Show main tube logs
  tube logs access    # Show nginx access logs
  tube logs error     # Show error logs
  tube logs -f        # Follow logs in real-time`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement logs functionality
			logType := "tube"
			if len(args) > 0 {
				logType = args[0]
			}
			cmd.Printf("Showing %s logs...\n", logType)
			return nil
		},
	}

	cmd.Flags().BoolP("follow", "f", false, "follow logs in real-time")
	cmd.Flags().IntP("lines", "n", 50, "number of lines to show")

	return cmd
}
