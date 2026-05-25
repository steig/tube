package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/steig/tube/internal/config"
	"go.yaml.in/yaml/v3"
)

// NewConfigCmd creates the config command
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage tube configuration",
		Long: `Manage tube configuration settings.

Examples:
  tube config set domain example.com
  tube config set ssl.enabled true
  tube config set proxy.dashboard_port 3249
  tube config set projects.myapp 3000
  tube config get domain
  tube config get projects
  tube config show`,
	}

	cmd.AddCommand(
		newConfigSetCmd(),
		newConfigGetCmd(),
		newConfigShowCmd(),
	)

	return cmd
}

// loadViperFromConfig opens the config file via viper so we can edit single keys
// without rewriting the full struct. Falls back to defaults if the file is missing.
func loadViperFromConfig(cmd *cobra.Command) (*viper.Viper, string, error) {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = config.ConfigPath()
	}

	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			if !os.IsNotExist(err) {
				return nil, configPath, fmt.Errorf("failed to read config: %w", err)
			}
		}
	}

	return v, configPath, nil
}

// parseValue coerces a CLI string into bool/int/string. Viper keeps whatever
// type we pass to Set, so this preserves YAML's native scalars.
func parseValue(s string) any {
	if b, err := strconv.ParseBool(s); err == nil && (s == "true" || s == "false") {
		return b
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return s
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value using dot-notation keys.

Examples:
  tube config set domain example.com
  tube config set ssl.enabled false
  tube config set projects.myapp 3000`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, raw := args[0], args[1]

			v, configPath, err := loadViperFromConfig(cmd)
			if err != nil {
				return err
			}

			value := parseValue(raw)
			v.Set(key, value)

			// Validate the resulting config before saving.
			var cfg config.Config
			if err := v.Unmarshal(&cfg); err != nil {
				return fmt.Errorf("failed to unmarshal updated config: %w", err)
			}
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("would produce invalid config: %w", err)
			}

			if err := v.WriteConfigAs(configPath); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			cmd.Printf("✓ Set %s = %v\n", key, value)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a configuration value using dot-notation keys.

Examples:
  tube config get domain
  tube config get ssl.enabled
  tube config get projects.myapp`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			v, _, err := loadViperFromConfig(cmd)
			if err != nil {
				return err
			}

			if !v.IsSet(key) {
				return fmt.Errorf("key %q is not set", key)
			}

			val := v.Get(key)

			// For maps and slices, render as YAML so nested values are readable.
			switch val.(type) {
			case map[string]any, []any:
				out, err := yaml.Marshal(val)
				if err != nil {
					return fmt.Errorf("failed to render value: %w", err)
				}
				cmd.Print(string(out))
			default:
				cmd.Println(val)
			}
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show full configuration",
		Long:  `Render the full effective configuration as YAML.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			out, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to marshal configuration: %w", err)
			}

			cmd.Println(strings.TrimRight(string(out), "\n"))
			return nil
		},
	}
}
