//go:build !windows

package service

import (
	"fmt"
	"os"
	"syscall"
)

// withFileLock acquires an exclusive flock on a per-service lock file, runs fn,
// and releases. This is what serializes Start/Stop *across separate tube
// processes* — the in-memory mutex only protects one process's goroutines.
//
// flock(2) is advisory and POSIX-only; this file is built for everything
// except windows. See lock_windows.go for the no-op fallback.
func (pm *ProcessManager) withFileLock(name string, fn func() error) error {
	f, err := os.OpenFile(pm.lockFilePath(name), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()

	return fn()
}
