package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

const installScriptURL = "https://raw.githubusercontent.com/steig/tube/main/scripts/install.sh"

// NewUpgradeCmd creates the upgrade command. It just re-runs the published
// install script — the same path users took the first time — so update
// logic only lives in one place.
func NewUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade tube to the latest release",
		Long: `Upgrade tube by re-running the published install script.

By default installs the latest release. Use --version to pin a specific
release, or --check to only show whether an upgrade is available without
installing anything.

Network and shell tools required: curl (or wget) and sh.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkOnly, _ := cmd.Flags().GetBool("check")
			version, _ := cmd.Flags().GetString("version")

			if checkOnly {
				return runCheck(cmd)
			}
			return runInstall(cmd, version)
		},
	}

	cmd.Flags().Bool("check", false, "only check for a newer version; don't install")
	cmd.Flags().String("version", "", "pin a specific version (e.g. v0.3.0)")
	return cmd
}

func runCheck(cmd *cobra.Command) error {
	// Lazy import via small inline call to avoid pulling update package into
	// every cobra command file.
	latest, current, newer, err := currentVsLatest(cmd)
	if err != nil {
		return err
	}
	if latest == "" {
		cmd.Println("Update check skipped (dev build or TUBE_NO_UPDATE_CHECK is set).")
		return nil
	}
	if newer {
		cmd.Printf("A newer release is available: %s (you have %s).\n", latest, current)
		cmd.Println("Run: tube upgrade")
	} else {
		cmd.Printf("tube %s is up to date.\n", current)
	}
	return nil
}

func runInstall(cmd *cobra.Command, version string) error {
	// Resolve the downloader once so we can give a clearer error than
	// `sh: curl: command not found` deep in the pipe.
	var (
		dl  *exec.Cmd
		hint string
	)
	switch {
	case haveBinary("curl"):
		dl = exec.Command("curl", "-fsSL", installScriptURL)
		hint = "curl"
	case haveBinary("wget"):
		dl = exec.Command("wget", "-qO-", installScriptURL)
		hint = "wget"
	default:
		return fmt.Errorf("need curl or wget on PATH to upgrade")
	}

	sh := exec.Command("sh")
	sh.Stdin, _ = dl.StdoutPipe()
	sh.Stdout = os.Stdout
	sh.Stderr = os.Stderr
	if version != "" {
		sh.Env = append(os.Environ(), "TUBE_VERSION="+version)
	}

	cmd.Printf("Fetching install script with %s and piping to sh...\n", hint)

	if err := sh.Start(); err != nil {
		return fmt.Errorf("failed to start sh: %w", err)
	}
	if err := dl.Run(); err != nil {
		return fmt.Errorf("failed to fetch install script: %w", err)
	}
	if err := sh.Wait(); err != nil {
		return fmt.Errorf("install script failed: %w", err)
	}
	return nil
}

func haveBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
