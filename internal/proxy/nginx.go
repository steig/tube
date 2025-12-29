package proxy

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/service"
)

// NginxManager manages nginx configuration generation and lifecycle
type NginxManager struct {
	config  *config.Config
	pm      *service.ProcessManager
	binary  string
	confDir string
}

// NewNginxManager creates a new NginxManager
func NewNginxManager(cfg *config.Config, pm *service.ProcessManager) (*NginxManager, error) {
	// Ensure nginx config directory exists
	if err := os.MkdirAll(cfg.Directories.Config, 0700); err != nil {
		return nil, fmt.Errorf("failed to create nginx config directory: %w", err)
	}

	return &NginxManager{
		config:  cfg,
		pm:      pm,
		binary:  "nginx",
		confDir: cfg.Directories.Config,
	}, nil
}

// GenerateConfig generates the nginx configuration from templates and projects
func (nm *NginxManager) GenerateConfig() (string, error) {
	// Read the main template
	mainTmpl, err := nm.readTemplate("main.conf.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to read main template: %w", err)
	}

	// Parse and execute the template
	tmpl, err := template.New("main").Parse(mainTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse main template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nm.config); err != nil {
		return "", fmt.Errorf("failed to execute main template: %w", err)
	}

	return buf.String(), nil
}

// GenerateProjectsConfig generates the projects-specific configuration
func (nm *NginxManager) GenerateProjectsConfig() (string, error) {
	// Read the projects template
	projTmpl, err := nm.readTemplate("projects.conf.tmpl")
	if err != nil {
		return "", fmt.Errorf("failed to read projects template: %w", err)
	}

	// Create template data with generation timestamp
	data := struct {
		*config.Config
		GeneratedAt string
	}{
		Config:      nm.config,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Parse and execute the template
	tmpl, err := template.New("projects").Parse(projTmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse projects template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute projects template: %w", err)
	}

	return buf.String(), nil
}

// WriteConfig writes the nginx configuration to disk
func (nm *NginxManager) WriteConfig() error {
	// Generate main config
	mainConfig, err := nm.GenerateConfig()
	if err != nil {
		return err
	}

	// Write main config
	mainPath := filepath.Join(nm.confDir, "nginx.conf")
	if err := os.WriteFile(mainPath, []byte(mainConfig), 0644); err != nil {
		return fmt.Errorf("failed to write main nginx config: %w", err)
	}

	// Generate projects config
	projConfig, err := nm.GenerateProjectsConfig()
	if err != nil {
		return err
	}

	// Write projects config
	projPath := filepath.Join(nm.confDir, "projects.conf")
	if err := os.WriteFile(projPath, []byte(projConfig), 0644); err != nil {
		return fmt.Errorf("failed to write projects nginx config: %w", err)
	}

	return nil
}

// TestConfig validates the nginx configuration by running `nginx -t`
func (nm *NginxManager) TestConfig() (bool, error) {
	mainPath := filepath.Join(nm.confDir, "nginx.conf")

	// Run nginx -t to test configuration
	cmd := exec.Command(nm.binary, "-t", "-c", mainPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("nginx configuration test failed:\n%s", stderr.String())
	}

	return true, nil
}

// Start starts the nginx service
func (nm *NginxManager) Start() error {
	// Test config first
	if ok, err := nm.TestConfig(); !ok || err != nil {
		return fmt.Errorf("nginx configuration invalid: %w", err)
	}

	// Start via ProcessManager
	if err := nm.pm.Start("nginx"); err != nil {
		return fmt.Errorf("failed to start nginx: %w", err)
	}

	return nil
}

// Stop stops the nginx service
func (nm *NginxManager) Stop() error {
	return nm.pm.Stop("nginx")
}

// Reload reloads the nginx configuration
func (nm *NginxManager) Reload() error {
	// Test config first
	if ok, err := nm.TestConfig(); !ok || err != nil {
		return fmt.Errorf("nginx configuration invalid: %w", err)
	}

	// Reload via ProcessManager
	if err := nm.pm.Reload("nginx"); err != nil {
		return fmt.Errorf("failed to reload nginx: %w", err)
	}

	return nil
}

// IsRunning checks if nginx is running
func (nm *NginxManager) IsRunning() (bool, error) {
	return nm.pm.IsRunning("nginx")
}

// Status returns the status of nginx
func (nm *NginxManager) Status() (string, error) {
	return nm.pm.Status("nginx")
}

// readTemplate reads a template file from the templates/nginx directory
func (nm *NginxManager) readTemplate(filename string) (string, error) {
	// Try multiple possible locations
	possiblePaths := []string{
		filepath.Join("templates", "nginx", filename),
		filepath.Join("/usr/local/etc", "tube", "nginx", filename),
		filepath.Join("/etc", "tube", "nginx", filename),
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
