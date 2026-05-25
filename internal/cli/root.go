package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/update"
)

// NewRootCmd creates the root command
func NewRootCmd(version, commit, date string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tube",
		Short: "Local development proxy with .test domains",
		Long: `Tube is a local development proxy that lets you:
- Access local projects via .test domains (e.g., myapp.test)
- Use HTTPS with automatic certificate management
- Monitor services via a web dashboard`,
		Version:       fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	// Add commands
	cmd.AddCommand(
		NewInitCmd(),
		NewSetupCmd(),
		NewUninstallCmd(),
		NewAddCmd(),
		NewRemoveCmd(),
		NewListCmd(),
		NewStartCmd(),
		NewStopCmd(),
		NewRestartCmd(),
		NewStatusCmd(),
		NewDNSStatusCmd(),
		NewConfigCmd(),
		NewLogsCmd(),
		NewDoctorCmd(),
		NewSSLCmd(),
		NewUpgradeCmd(),
	)

	// Global flags
	cmd.PersistentFlags().StringP("config", "c", "", "path to config file (default: ~/.tube/config.yaml)")

	// After every successful command, print at most one line nudging the user
	// to upgrade if a newer release exists. Quiet by default — skipped on
	// non-TTY stdout, on upgrade/version/--help itself, when TUBE_NO_UPDATE_CHECK
	// is set, and on dev builds.
	cmd.PersistentPostRun = func(c *cobra.Command, args []string) {
		update.Once.Do(func() {
			maybePrintUpdateNotice(c, version)
		})
	}

	return cmd
}

func maybePrintUpdateNotice(c *cobra.Command, currentVersion string) {
	// Skip if stderr isn't a terminal — we don't want the notice ending up
	// in script output, piped JSON, CI logs, etc.
	if fi, err := os.Stderr.Stat(); err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return
	}
	// Don't pile a notice onto the upgrade command itself.
	if c.Name() == "upgrade" || c.Name() == "help" || c.Name() == "version" {
		return
	}

	r, err := update.Check(currentVersion, update.DefaultCachePath())
	if err != nil || !r.Newer {
		return
	}
	fmt.Fprintf(os.Stderr, "\nA new tube release is available: %s (you have %s). Run: tube upgrade\n",
		r.Latest, r.Current)
}
