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

// DnsmasqManager manages dnsmasq configuration generation and lifecycle
type DnsmasqManager struct {
	config  *config.Config
	pm      *service.ProcessManager
	binary  string
	confDir string
}

// NewDnsmasqManager creates a new DnsmasqManager
func NewDnsmasqManager(cfg *config.Config, pm *service.ProcessManager) (*DnsmasqManager, error) {
	// Ensure dnsmasq config directory exists
	if err := os.MkdirAll(cfg.Directories.Config, 0700); err != nil {
		return nil, fmt.Errorf("failed to create dnsmasq config directory: %w", err)
	}

	return &DnsmasqManager{
		config:  cfg,
		pm:      pm,
		binary:  "dnsmasq",
		confDir: cfg.Directories.Config,
	}, nil
}

// GenerateConfig generates the dnsmasq configuration from templates
func (dm *DnsmasqManager) GenerateConfig() (string, error) {
	// Read the template
	tmplContent, err := dm.readTemplate("dnsmasq.conf.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to read dnsmasq template: %w", err)
	}

	// Parse and execute the template
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

// WriteConfig writes the dnsmasq configuration to disk
func (dm *DnsmasqManager) WriteConfig() error {
	config, err := dm.GenerateConfig()
	if err != nil {
		return err
	}

	confPath := filepath.Join(dm.confDir, "dnsmasq.conf")
	if err := os.WriteFile(confPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write dnsmasq config: %w", err)
	}

	return nil
}

// Start starts the dnsmasq service
func (dm *DnsmasqManager) Start() error {
	// dnsmasq will read the config file automatically
	// We don't need to pass arguments if the file is in the expected location
	if err := dm.pm.Start("dnsmasq"); err != nil {
		return fmt.Errorf("failed to start dnsmasq: %w", err)
	}

	return nil
}

// Stop stops the dnsmasq service
func (dm *DnsmasqManager) Stop() error {
	return dm.pm.Stop("dnsmasq")
}

// IsRunning checks if dnsmasq is running
func (dm *DnsmasqManager) IsRunning() (bool, error) {
	return dm.pm.IsRunning("dnsmasq")
}

// Status returns the status of dnsmasq
func (dm *DnsmasqManager) Status() (string, error) {
	return dm.pm.Status("dnsmasq")
}

// readTemplate reads a template file from the templates/dnsmasq directory
func (dm *DnsmasqManager) readTemplate(filename string) (string, error) {
	// Try multiple possible locations
	possiblePaths := []string{
		filepath.Join("templates", "dnsmasq", filename),
		filepath.Join("/usr/local/etc", "tube", "dnsmasq", filename),
		filepath.Join("/etc", "tube", "dnsmasq", filename),
	}

	var lastErr error
	for _, path := range possiblePaths {
		content, err := os.ReadFile(path)
		if err == nil {
			return string(content), nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("failed to read template %s: %w", filename, lastErr)
}
