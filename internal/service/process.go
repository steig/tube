package service

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Start starts the named service by spawning the appropriate process.
// Both the in-memory mutex and a file-based flock are acquired so that two
// independent tube processes (e.g. CLI + tube-gui) cannot double-spawn nginx.
func (pm *ProcessManager) Start(name string) error {
	return pm.withFileLock(name, func() error {
		pm.mu.Lock()
		defer pm.mu.Unlock()
		return pm.startLocked(name)
	})
}

func (pm *ProcessManager) startLocked(name string) error {
	if isRunning, err := pm.isRunningLocked(name); err == nil && isRunning {
		return fmt.Errorf("service %s is already running", name)
	}

	binary, args, err := pm.getServiceConfig(name)
	if err != nil {
		return err
	}

	cmd := exec.Command(binary, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", name, err)
	}

	if err := pm.writePID(name, cmd.Process.Pid); err != nil {
		_ = cmd.Process.Kill()
		return err
	}

	return nil
}

// Stop stops the named service gracefully.
// Sends SIGTERM, waits 5s, then SIGKILL if the process is still alive.
func (pm *ProcessManager) Stop(name string) error {
	return pm.withFileLock(name, func() error {
		pm.mu.Lock()
		defer pm.mu.Unlock()
		return pm.stopLocked(name)
	})
}

func (pm *ProcessManager) stopLocked(name string) error {
	pid, err := pm.readPID(name)
	if err != nil {
		return err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		_ = pm.cleanupPID(name)
		return fmt.Errorf("service %s (pid %d) not found: %w", name, pid, err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM to %s: %w", name, err)
	}

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			if err := process.Signal(syscall.SIGKILL); err != nil {
				_ = pm.cleanupPID(name)
				return fmt.Errorf("failed to force kill %s: %w", name, err)
			}
			time.Sleep(100 * time.Millisecond)
			_ = pm.cleanupPID(name)
			return nil

		case <-ticker.C:
			if err := process.Signal(syscall.Signal(0)); err != nil {
				_ = pm.cleanupPID(name)
				return nil
			}
		}
	}
}

// Reload reloads the named service by sending SIGHUP
// This works for services like nginx that support graceful reload
func (pm *ProcessManager) Reload(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pid, err := pm.readPID(name)
	if err != nil {
		return err
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		_ = pm.cleanupPID(name)
		return fmt.Errorf("service %s (pid %d) not found: %w", name, pid, err)
	}

	if err := process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("failed to send SIGHUP to %s: %w", name, err)
	}

	return nil
}

// Status returns a string describing the status of the service
func (pm *ProcessManager) Status(name string) (string, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	isRunning, err := pm.isRunningLocked(name)
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
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.isRunningLocked(name)
}

func (pm *ProcessManager) isRunningLocked(name string) (bool, error) {
	pid, err := pm.readPID(name)
	if err != nil {
		return false, nil // Service not running if no PID file
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		_ = pm.cleanupPID(name)
		return false, nil
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		_ = pm.cleanupPID(name)
		return false, nil
	}

	return true, nil
}

// errNotRunning is returned by stopLocked when the service has no PID file.
// Callers (StopAll) match on this with errors.Is rather than fragile string
// comparison.
var errNotRunning = fmt.Errorf("service not running")

// getServiceConfig returns the binary + args for a service. Callers can override
// the defaults via SetServiceConfig (e.g. dnsmasq -C path).
func (pm *ProcessManager) getServiceConfig(name string) (string, []string, error) {
	if cfg, ok := pm.configs[name]; ok {
		return cfg.Binary, cfg.Args, nil
	}
	switch name {
	case "nginx":
		return "nginx", []string{"-g", "daemon off;"}, nil
	case "dnsmasq":
		return "dnsmasq", []string{}, nil
	default:
		return "", nil, fmt.Errorf("unknown service: %s", name)
	}
}

// StopAll stops every known service. Services that aren't running are not errors.
func (pm *ProcessManager) StopAll() error {
	var errs []string
	for _, svc := range knownServices {
		if err := pm.Stop(svc); err != nil && !errors.Is(err, errNotRunning) {
			errs = append(errs, fmt.Sprintf("%s: %v", svc, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors stopping services:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// StartAll starts every known service. If any service fails to start, all
// previously-started services are rolled back.
func (pm *ProcessManager) StartAll() error {
	var errs []string
	for _, svc := range knownServices {
		if err := pm.Start(svc); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", svc, err))
		}
	}
	if len(errs) > 0 {
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
	pm.mu.Lock()
	defer pm.mu.Unlock()

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
