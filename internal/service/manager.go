package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// knownServices is the canonical list of services tube manages.
// Add new services here and StartAll/StopAll/RestartAll pick them up automatically.
var knownServices = []string{"nginx", "dnsmasq"}

// ProcessManager manages service lifecycle by spawning processes directly
// (not via systemd) so it can run in a per-user dev context.
//
// All state-mutating operations (Start, Stop, Reload, IsRunning, Status) acquire
// mu to serialize PID-file I/O. Multiple goroutines (CLI + dashboard HTTP handlers)
// may invoke these concurrently.
type ProcessManager struct {
	mu      sync.Mutex
	pidDir  string         // Directory where PID files are stored (~/.tube/pids/)
	configs map[string]ServiceConfig
}

// ServiceConfig describes how to start a single service.
type ServiceConfig struct {
	Binary string
	Args   []string
}

// NewProcessManager creates a new ProcessManager with the specified PID directory
func NewProcessManager(pidDir string) (*ProcessManager, error) {
	// Ensure PID directory exists
	if err := os.MkdirAll(pidDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create pid directory: %w", err)
	}

	return &ProcessManager{
		pidDir:  pidDir,
		configs: map[string]ServiceConfig{},
	}, nil
}

// SetServiceConfig overrides the spawn config for a service (binary + args).
// Callers (e.g. proxy.DnsmasqManager) use this to pass per-instance flags like
// dnsmasq's -C config-file argument.
func (pm *ProcessManager) SetServiceConfig(name string, cfg ServiceConfig) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.configs[name] = cfg
}

// lockFilePath returns the path of the per-service lock file used by
// the platform-specific withFileLock implementations.
func (pm *ProcessManager) lockFilePath(name string) string {
	return filepath.Join(pm.pidDir, name+".lock")
}

// pidFilePath returns the path to the PID file for a service
func (pm *ProcessManager) pidFilePath(name string) string {
	return filepath.Join(pm.pidDir, name+".pid")
}

// writePID writes the given PID to a file for the service
func (pm *ProcessManager) writePID(name string, pid int) error {
	pidFile := pm.pidFilePath(name)
	if err := os.WriteFile(pidFile, fmt.Appendf(nil, "%d\n", pid), 0600); err != nil {
		return fmt.Errorf("failed to write pid file for %s: %w", name, err)
	}
	return nil
}

// readPID reads the PID from a file for the service.
// Returns an error wrapping errNotRunning if the PID file is absent.
func (pm *ProcessManager) readPID(name string) (int, error) {
	pidFile := pm.pidFilePath(name)
	content, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("service %s: %w", name, errNotRunning)
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

