package cli

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/proxy"
	"github.com/steig/tube/internal/service"
)

// ProjectStatus represents the status of a project
type ProjectStatus struct {
	Name    string
	Port    int
	Running bool
	LocalURL string
}

// ValidateProjectName validates that a project name is DNS-safe
func ValidateProjectName(name string) error {
	// Check if empty
	if name == "" {
		return fmt.Errorf("project name cannot be empty")
	}

	// Check max length (DNS label limit)
	if len(name) > 63 {
		return fmt.Errorf("project name must be at most 63 characters (got %d)", len(name))
	}

	// Check valid characters (alphanumeric and hyphens only)
	if !regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`).MatchString(name) {
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

// IsPortListening checks if a port is currently listening
func IsPortListening(port int) (bool, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		// If we can't listen, the port is likely in use
		if strings.Contains(err.Error(), "address already in use") {
			return true, nil
		}
		return false, err
	}
	_ = listener.Close()

	return false, nil
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

	// If services are running, reload them
	if isRunning, _ := pm.IsRunning("nginx"); isRunning {
		if err := ngx.Reload(); err != nil {
			return fmt.Errorf("failed to reload nginx: %w", err)
		}
	}

	if isRunning, _ := pm.IsRunning("dnsmasq"); isRunning {
		if err := dms.Stop(); err != nil {
			return fmt.Errorf("failed to restart dnsmasq: %w", err)
		}
		if err := dms.Start(); err != nil {
			return fmt.Errorf("failed to restart dnsmasq: %w", err)
		}
	}

	return nil
}

// RemoveProject removes a project from the configuration
func RemoveProject(cfg *config.Config, configPath string, pm *service.ProcessManager, ngx *proxy.NginxManager, dms *proxy.DnsmasqManager, name string) error {
	// Check if project exists
	if _, exists := cfg.Projects[name]; !exists {
		return fmt.Errorf("project %q not found", name)
	}

	// Remove the project
	delete(cfg.Projects, name)

	// Save config
	if err := cfg.Save(configPath); err != nil {
		// Re-add the project if save fails
		cfg.Projects[name] = 0 // Will be refilled by reload
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Regenerate nginx config
	if err := ngx.WriteConfig(); err != nil {
		return fmt.Errorf("failed to generate nginx configuration: %w", err)
	}

	// Regenerate dnsmasq config
	if err := dms.WriteConfig(); err != nil {
		return fmt.Errorf("failed to generate dnsmasq configuration: %w", err)
	}

	// If services are running, reload them
	if isRunning, _ := pm.IsRunning("nginx"); isRunning {
		if err := ngx.Reload(); err != nil {
			return fmt.Errorf("failed to reload nginx: %w", err)
		}
	}

	if isRunning, _ := pm.IsRunning("dnsmasq"); isRunning {
		if err := dms.Stop(); err != nil {
			return fmt.Errorf("failed to restart dnsmasq: %w", err)
		}
		if err := dms.Start(); err != nil {
			return fmt.Errorf("failed to restart dnsmasq: %w", err)
		}
	}

	return nil
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
