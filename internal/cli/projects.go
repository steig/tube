package cli

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/proxy"
	"github.com/steig/tube/internal/service"
)

// projectNameRe matches DNS-safe labels: alphanumeric with internal hyphens.
// Compiled once at package load to avoid recompiling per ValidateProjectName call.
var projectNameRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// ProjectStatus represents the status of a project
type ProjectStatus struct {
	Name     string
	Port     int
	Running  bool
	LocalURL string
}

// ValidateProjectName validates that a project name is DNS-safe
func ValidateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if len(name) > 63 {
		return fmt.Errorf("project name must be at most 63 characters (got %d)", len(name))
	}
	if !projectNameRe.MatchString(name) {
		return fmt.Errorf("project name can only contain alphanumeric characters and hyphens, and must start/end with alphanumeric")
	}
	return nil
}

// ValidatePort validates that a port number is valid
func ValidatePort(port int) error {
	// Check port range
	if port < 1024 || port > 65535 {
		return fmt.Errorf("port must be between 1024 and 65535 (got %d)", port)
	}

	return nil
}

// IsPortListening reports whether anything is accepting connections on the
// given localhost port. Uses a short-timeout dial instead of trying to bind
// the port ourselves — binding is racy, OS-string-dependent, and on macOS can
// trigger firewall prompts.
func IsPortListening(port int) (bool, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
	if err != nil {
		return false, nil
	}
	_ = conn.Close()
	return true, nil
}

// AddProject adds a new project to the configuration
func AddProject(cfg *config.Config, configPath string, pm *service.ProcessManager, ngx *proxy.NginxManager, dms *proxy.DnsmasqManager, name string, port int) error {
	// Validate inputs
	if err := ValidateProjectName(name); err != nil {
		return err
	}

	if err := ValidatePort(port); err != nil {
		return err
	}

	// Check if project already exists
	if _, exists := cfg.Projects[name]; exists {
		return fmt.Errorf("project %q already exists", name)
	}

	// Check if port is already used by another project
	for projectName, projectPort := range cfg.Projects {
		if projectPort == port {
			return fmt.Errorf("port %d is already used by project %q", port, projectName)
		}
	}

	// Add the project
	cfg.Projects[name] = port

	// Save config
	if err := cfg.Save(configPath); err != nil {
		// Remove the project if save fails
		delete(cfg.Projects, name)
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Regenerate nginx config
	if err := ngx.WriteConfig(); err != nil {
		// Remove the project if config generation fails
		delete(cfg.Projects, name)
		_ = cfg.Save(configPath)
		return fmt.Errorf("failed to generate nginx configuration: %w", err)
	}

	// Regenerate dnsmasq config
	if err := dms.WriteConfig(); err != nil {
		// Remove the project if config generation fails
		delete(cfg.Projects, name)
		_ = cfg.Save(configPath)
		return fmt.Errorf("failed to generate dnsmasq configuration: %w", err)
	}

	return reloadIfRunning(pm, ngx, dms)
}

// reloadIfRunning issues a SIGHUP reload to any service that's running so the
// freshly-written config picks up without a service interruption. Both nginx
// and dnsmasq accept SIGHUP for config reload — the old code stop+start'd
// dnsmasq which left a brief gap with no .test resolution.
func reloadIfRunning(pm *service.ProcessManager, ngx *proxy.NginxManager, dms *proxy.DnsmasqManager) error {
	if running, _ := pm.IsRunning("nginx"); running {
		if err := ngx.Reload(); err != nil {
			return fmt.Errorf("failed to reload nginx: %w", err)
		}
	}
	if running, _ := pm.IsRunning("dnsmasq"); running {
		if err := dms.Reload(); err != nil {
			return fmt.Errorf("failed to reload dnsmasq: %w", err)
		}
	}
	return nil
}

// RemoveProject removes a project from the configuration
func RemoveProject(cfg *config.Config, configPath string, pm *service.ProcessManager, ngx *proxy.NginxManager, dms *proxy.DnsmasqManager, name string) error {
	port, exists := cfg.Projects[name]
	if !exists {
		return fmt.Errorf("project %q not found", name)
	}

	delete(cfg.Projects, name)

	if err := cfg.Save(configPath); err != nil {
		// Restore the original port — not zero — so in-memory state matches what's on disk.
		cfg.Projects[name] = port
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	if err := ngx.WriteConfig(); err != nil {
		return fmt.Errorf("failed to generate nginx configuration: %w", err)
	}
	if err := dms.WriteConfig(); err != nil {
		return fmt.Errorf("failed to generate dnsmasq configuration: %w", err)
	}

	return reloadIfRunning(pm, ngx, dms)
}

// ListProjects returns a list of all projects with their status
func ListProjects(cfg *config.Config) ([]ProjectStatus, error) {
	var statuses []ProjectStatus

	for name, port := range cfg.Projects {
		isListening, err := IsPortListening(port)
		if err != nil {
			// Log but continue
			isListening = false
		}

		// Handle LocalDomain with or without leading dot
		domain := strings.TrimPrefix(cfg.Proxy.LocalDomain, ".")

		// Use https:// scheme when SSL is enabled
		scheme := "http"
		if cfg.SSL.Enabled {
			scheme = "https"
		}

		status := ProjectStatus{
			Name:    name,
			Port:    port,
			Running: isListening,
			LocalURL: fmt.Sprintf("%s://%s.%s", scheme, name, domain),
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetProjectPort gets the port for a specific project
func GetProjectPort(cfg *config.Config, name string) (int, error) {
	port, exists := cfg.Projects[name]
	if !exists {
		return 0, fmt.Errorf("project %q not found", name)
	}
	return port, nil
}

// ProjectExists checks if a project exists
func ProjectExists(cfg *config.Config, name string) bool {
	_, exists := cfg.Projects[name]
	return exists
}

// PortExists checks if a port is already used
func PortExists(cfg *config.Config, port int) bool {
	for _, p := range cfg.Projects {
		if p == port {
			return true
		}
	}
	return false
}

// ParsePort parses a port string and validates it
func ParsePort(portStr string) (int, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port number: %s", portStr)
	}

	if err := ValidatePort(port); err != nil {
		return 0, err
	}

	return port, nil
}
