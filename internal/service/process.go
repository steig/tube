package service

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Start starts the named service by spawning the appropriate process
// It supports "nginx" and "dnsmasq" services
func (pm *ProcessManager) Start(name string) error {
	// Check if already running
	isRunning, err := pm.IsRunning(name)
	if err == nil && isRunning {
		return fmt.Errorf("service %s is already running", name)
	}

	// Get the binary path and arguments for the service
	binary, args, err := pm.getServiceConfig(name)
	if err != nil {
		return err
	}

	// Create the command
	cmd := exec.Command(binary, args...)

	// Suppress stdout/stderr for background services
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start the process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", name, err)
	}

	// Write PID file
	if err := pm.writePID(name, cmd.Process.Pid); err != nil {
		// Try to kill the process if we can't write the PID file
		_ = cmd.Process.Kill()
		return err
	}

	return nil
}

// Stop stops the named service gracefully
// It sends SIGTERM, waits a timeout, then sends SIGKILL if needed
func (pm *ProcessManager) Stop(name string) error {
	// Read the PID from file
	pid, err := pm.readPID(name)
	if err != nil {
		return err
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		// Clean up the stale PID file
		_ = pm.cleanupPID(name)
		return fmt.Errorf("service %s (pid %d) not found: %w", name, pid, err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to %s: %w", name, err)
	}

	// Wait for process to terminate (with timeout)
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			// Process didn't terminate gracefully, force kill it
			if err := process.Signal(syscall.SIGKILL); err != nil {
				// Process may already be dead
				_ = pm.cleanupPID(name)
				return fmt.Errorf("failed to force kill %s: %w", name, err)
			}
			// Wait a bit for SIGKILL to take effect
			time.Sleep(100 * time.Millisecond)
			_ = pm.cleanupPID(name)
			return nil

		case <-ticker.C:
			// Check if process is still running
			// We do this by trying to send signal 0 (which doesn't kill but checks existence)
			if err := process.Signal(syscall.Signal(0)); err != nil {
				// Process is dead
				_ = pm.cleanupPID(name)
				return nil
			}
		}
	}
}

// Reload reloads the named service by sending SIGHUP
// This works for services like nginx that support graceful reload
func (pm *ProcessManager) Reload(name string) error {
	// Read the PID from file
	pid, err := pm.readPID(name)
	if err != nil {
		return err
	}

	// Find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		// Clean up the stale PID file
		_ = pm.cleanupPID(name)
		return fmt.Errorf("service %s (pid %d) not found: %w", name, pid, err)
	}

	// Send SIGHUP to reload configuration
	if err := process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send SIGHUP to %s: %w", name, err)
	}

	return nil
}

// Status returns a string describing the status of the service
func (pm *ProcessManager) Status(name string) (string, error) {
	isRunning, err := pm.IsRunning(name)
	if err != nil {
		return "unknown", err
	}

	if isRunning {
		pid, _ := pm.readPID(name)
		return fmt.Sprintf("running (pid %d)", pid), nil
	}

	return "stopped", nil
}

// IsRunning checks if the named service is currently running
func (pm *ProcessManager) IsRunning(name string) (bool, error) {
	// Try to read PID file
	pid, err := pm.readPID(name)
	if err != nil {
		return false, nil // Service not running if no PID file
	}

	// Try to find the process
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist, clean up stale PID file
		_ = pm.cleanupPID(name)
		return false, nil
	}

	// Check if process still exists by sending signal 0
	if err := process.Signal(syscall.Signal(0)); err != nil {
		// Process is dead, clean up stale PID file
		_ = pm.cleanupPID(name)
		return false, nil
	}

	return true, nil
}

// getServiceConfig returns the binary path and arguments for a service
func (pm *ProcessManager) getServiceConfig(name string) (string, []string, error) {
	switch name {
	case "nginx":
		return "nginx", []string{"-g", "daemon off;"}, nil

	case "dnsmasq":
		return "dnsmasq", []string{}, nil

	default:
		return "", nil, fmt.Errorf("unknown service: %s", name)
	}
}

// StopAll stops all known services
func (pm *ProcessManager) StopAll() error {
	services := []string{"nginx", "dnsmasq"}
	var errs []string

	for _, service := range services {
		if err := pm.Stop(service); err != nil {
			// Check if it's just not running
			if !strings.Contains(err.Error(), "is not running") {
				errs = append(errs, fmt.Sprintf("%s: %v", service, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping services:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// StartAll starts all known services
func (pm *ProcessManager) StartAll() error {
	services := []string{"nginx", "dnsmasq"}
	var errs []string

	for _, service := range services {
		if err := pm.Start(service); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", service, err))
		}
	}

	if len(errs) > 0 {
		// Try to stop all services if startup failed
		_ = pm.StopAll()
		return fmt.Errorf("errors starting services:\n  - %s", strings.Join(errs, "\n  - "))
	}

	return nil
}

// RestartAll restarts all services
func (pm *ProcessManager) RestartAll() error {
	if err := pm.StopAll(); err != nil {
		// Log but continue
		fmt.Printf("Warning: errors stopping services: %v\n", err)
	}

	time.Sleep(500 * time.Millisecond) // Give services time to fully stop

	return pm.StartAll()
}

// GetServicePID returns the PID of a service, or 0 if not running
func (pm *ProcessManager) GetServicePID(name string) int {
	pid, err := pm.readPID(name)
	if err != nil {
		return 0
	}
	return pid
}

// SetupSignalHandling sets up signal handlers to gracefully stop services
// This should be called once at application startup
func (pm *ProcessManager) SetupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		fmt.Printf("Received signal %v, stopping services...\n", sig)
		_ = pm.StopAll()
		os.Exit(0)
	}()
}
