package proxy

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/service"
)

// DnsmasqManager manages dnsmasq configuration generation and lifecycle.
type DnsmasqManager struct {
	config  *config.Config
	pm      *service.ProcessManager
	binary  string
	confDir string
}

// NewDnsmasqManager creates a new DnsmasqManager. It also tells the
// ProcessManager to spawn dnsmasq with -C pointing at the config tube wrote —
// without this, dnsmasq would silently read /usr/local/etc/dnsmasq.conf
// instead of tube's generated config.
func NewDnsmasqManager(cfg *config.Config, pm *service.ProcessManager) (*DnsmasqManager, error) {
	if err := os.MkdirAll(cfg.Directories.Config, 0700); err != nil {
		return nil, fmt.Errorf("failed to create dnsmasq config directory: %w", err)
	}

	confPath := filepath.Join(cfg.Directories.Config, "dnsmasq.conf")
	pm.SetServiceConfig("dnsmasq", service.ServiceConfig{
		Binary: "dnsmasq",
		// -k keeps dnsmasq in the foreground so the PID we recorded matches
		//    the running process (otherwise it daemonizes and forks).
		// -C points at tube's generated config so we never inherit a stale
		//    system-wide /usr/local/etc/dnsmasq.conf.
		Args: []string{"-k", "-C", confPath},
	})

	return &DnsmasqManager{
		config:  cfg,
		pm:      pm,
		binary:  "dnsmasq",
		confDir: cfg.Directories.Config,
	}, nil
}

// GenerateConfig generates the dnsmasq configuration from templates.
func (dm *DnsmasqManager) GenerateConfig() (string, error) {
	tmplContent, err := readTemplate("dnsmasq", "dnsmasq.conf.tmpl")
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("dnsmasq").Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse dnsmasq template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, dm.config); err != nil {
		return "", fmt.Errorf("failed to execute dnsmasq template: %w", err)
	}
	return buf.String(), nil
}

// WriteConfig writes the dnsmasq configuration to disk.
func (dm *DnsmasqManager) WriteConfig() error {
	cfg, err := dm.GenerateConfig()
	if err != nil {
		return err
	}
	confPath := filepath.Join(dm.confDir, "dnsmasq.conf")
	if err := os.WriteFile(confPath, []byte(cfg), 0644); err != nil {
		return fmt.Errorf("failed to write dnsmasq config: %w", err)
	}
	return nil
}

// Start starts the dnsmasq service.
func (dm *DnsmasqManager) Start() error {
	if err := dm.pm.Start("dnsmasq"); err != nil {
		return fmt.Errorf("failed to start dnsmasq: %w", err)
	}
	return nil
}

// Stop stops the dnsmasq service.
func (dm *DnsmasqManager) Stop() error { return dm.pm.Stop("dnsmasq") }

// Reload sends SIGHUP so dnsmasq re-reads its config without restarting,
// avoiding a gap in name resolution.
func (dm *DnsmasqManager) Reload() error { return dm.pm.Reload("dnsmasq") }

// IsRunning checks if dnsmasq is running.
func (dm *DnsmasqManager) IsRunning() (bool, error) { return dm.pm.IsRunning("dnsmasq") }

// Status returns the status of dnsmasq.
func (dm *DnsmasqManager) Status() (string, error) { return dm.pm.Status("dnsmasq") }
