package cli

import (
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the config command
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage tube configuration",
		Long: `Manage tube configuration settings.

Examples:
  tube config set domain example.com
  tube config set tunnel-prefix dev-
  tube config get domain
  tube config show`,
	}

	cmd.AddCommand(
		newConfigSetCmd(),
		newConfigGetCmd(),
		newConfigShowCmd(),
	)

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement config set
			cmd.Printf("✓ Set %s = %s\n", args[0], args[1])
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement config get
			cmd.Printf("value for %s\n", args[0])
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show full configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement config show
			cmd.Println("Current configuration:")
			return nil
		},
	}
}
