package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/dns"
	"github.com/steig/tube/internal/service"
)

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// check is one row of the doctor report.
type check struct {
	name   string
	ok     bool
	detail string
	hint   string // install/fix command, shown only when !ok
}

// NewDoctorCmd creates the doctor command
func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check dependencies and system setup",
		Long: `Diagnose tube installation. Checks required binaries, DNS resolver
configuration, SSL certificate state, and running services.

Exits non-zero if any required check fails.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, cfgErr := loadCfg(cmd)

			cmd.Println("Checking tube setup...")
			cmd.Println()

			var checks []check
			var failed bool

			// --- Config
			if cfgErr != nil {
				checks = append(checks, check{
					name:   "config",
					ok:     false,
					detail: cfgErr.Error(),
					hint:   "run: tube init",
				})
				// Print what we have, then bail.
				printChecks(cmd, checks)
				return fmt.Errorf("config did not load")
			}
			checks = append(checks, check{
				name:   "config",
				ok:     true,
				detail: "loaded and valid",
			})

			// --- Binaries
			checks = append(checks, binaryCheck("nginx", cfg.Nginx.Binary, "brew install nginx", true))
			checks = append(checks, binaryCheck("dnsmasq", cfg.Dnsmasq.Binary, "brew install dnsmasq", true))
			if cfg.SSL.Enabled {
				checks = append(checks, binaryCheck("mkcert", cfg.SSL.MkcertBinary, "brew install mkcert", true))
			}
			// cloudflared is optional — tunnel feature isn't wired up yet.
			checks = append(checks, binaryCheck("cloudflared", cfg.Tunnel.Binary, "brew install cloudflared", false))

			// --- DNS resolver
			rm := dns.NewResolverManager(cfg.Proxy.LocalDomain, "127.0.0.1")
			if status, err := rm.Status(); err != nil {
				checks = append(checks, check{
					name:   "DNS resolver",
					ok:     false,
					detail: err.Error(),
					hint:   "run: tube setup",
				})
			} else if strings.HasPrefix(status, "configured") {
				checks = append(checks, check{name: "DNS resolver", ok: true, detail: status})
			} else {
				checks = append(checks, check{
					name:   "DNS resolver",
					ok:     false,
					detail: status,
					hint:   "run: tube setup",
				})
			}

			// --- SSL certs
			if cfg.SSL.Enabled {
				if fileExists(cfg.SSL.CertFile) && fileExists(cfg.SSL.KeyFile) {
					checks = append(checks, check{name: "SSL certificate", ok: true, detail: cfg.SSL.CertFile})
				} else {
					checks = append(checks, check{
						name:   "SSL certificate",
						ok:     false,
						detail: "missing cert or key",
						hint:   "run: tube ssl generate",
					})
				}
			}

			// --- Services
			pm, err := service.NewProcessManager(cfg.Directories.PIDs)
			if err == nil {
				for _, svc := range []string{"nginx", "dnsmasq"} {
					status, _ := pm.Status(svc)
					// Service status is informational only — not a failure.
					checks = append(checks, check{
						name:   "service " + svc,
						ok:     strings.HasPrefix(status, "running"),
						detail: status,
					})
				}
			}

			// Render
			for _, c := range checks {
				if !c.ok && requiredCheck(c.name) {
					failed = true
				}
			}
			printChecks(cmd, checks)

			if failed {
				return fmt.Errorf("one or more required checks failed")
			}
			return nil
		},
	}
}

// requiredCheck returns true when a failing check should cause non-zero exit.
// Service status (running/stopped) and optional binaries (cloudflared) are advisory.
func requiredCheck(name string) bool {
	if strings.HasPrefix(name, "service ") {
		return false
	}
	if name == "cloudflared" {
		return false
	}
	return true
}

func binaryCheck(label, binary, hint string, required bool) check {
	path, err := exec.LookPath(binary)
	if err != nil {
		c := check{name: label, ok: false, detail: "not found in PATH"}
		if required {
			c.hint = hint
		} else {
			c.detail = "not found (optional)"
		}
		return c
	}
	return check{name: label, ok: true, detail: path}
}

func printChecks(cmd *cobra.Command, checks []check) {
	const labelWidth = 18
	var hints []string

	for _, c := range checks {
		mark := "✓"
		if !c.ok {
			mark = "✗"
			if !requiredCheck(c.name) {
				mark = "○"
			}
		}
		cmd.Printf("  %s %-*s %s\n", mark, labelWidth, c.name, c.detail)
		if !c.ok && c.hint != "" {
			hints = append(hints, fmt.Sprintf("  - %s: %s", c.name, c.hint))
		}
	}

	if len(hints) > 0 {
		cmd.Println()
		cmd.Println("To fix:")
		for _, h := range hints {
			cmd.Println(h)
		}
	}
}
