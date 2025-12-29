package service

import (
	"fmt"
	"os"
	"path/filepath"
)

// ServiceManager defines the interface for managing system services
// It provides a platform-agnostic way to start, stop, and check the status
// of system services like nginx and dnsmasq.
type ServiceManager interface {
	// Start starts the named service
	Start(name string) error

	// Stop stops the named service
	Stop(name string) error

	// Status returns the status of the named service
	Status(name string) (string, error)

	// IsRunning checks if the named service is currently running
	IsRunning(name string) (bool, error)
}

// ProcessManager implements ServiceManager using direct process spawning.
// It manages service lifecycle by spawning processes directly (not using systemd)
// to support cross-platform development environments.
type ProcessManager struct {
	pidDir string // Directory where PID files are stored (~/.tube/pids/)
	// Config would be added here later if needed
}

// NewProcessManager creates a new ProcessManager with the specified PID directory
func NewProcessManager(pidDir string) (*ProcessManager, error) {
	// Ensure PID directory exists
	if err := os.MkdirAll(pidDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create pid directory: %w", err)
	}

	return &ProcessManager{
		pidDir: pidDir,
	}, nil
}

// pidFilePath returns the path to the PID file for a service
func (pm *ProcessManager) pidFilePath(name string) string {
	return filepath.Join(pm.pidDir, name+".pid")
}

// writePID writes the given PID to a file for the service
func (pm *ProcessManager) writePID(name string, pid int) error {
	pidFile := pm.pidFilePath(name)
	content := []byte(fmt.Sprintf("%d\n", pid))

	if err := os.WriteFile(pidFile, content, 0600); err != nil {
		return fmt.Errorf("failed to write pid file for %s: %w", name, err)
	}

	return nil
}

// readPID reads the PID from a file for the service
func (pm *ProcessManager) readPID(name string) (int, error) {
	pidFile := pm.pidFilePath(name)
	content, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("service %s is not running (no pid file)", name)
		}
		return 0, fmt.Errorf("failed to read pid file for %s: %w", name, err)
	}

	var pid int
	_, err = fmt.Sscanf(string(content), "%d", &pid)
	if err != nil {
		return 0, fmt.Errorf("invalid pid file for %s: %w", name, err)
	}

	return pid, nil
}

// cleanupPID removes the PID file for a service
func (pm *ProcessManager) cleanupPID(name string) error {
	pidFile := pm.pidFilePath(name)
	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove pid file for %s: %w", name, err)
	}
	return nil
}

// Verify that ProcessManager implements ServiceManager
var _ ServiceManager = (*ProcessManager)(nil)
