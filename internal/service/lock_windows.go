//go:build windows

package service

// withFileLock on Windows is a no-op: cross-process serialization would
// require CreateMutex/LockFileEx and the rest of the runtime stack (nginx,
// dnsmasq, mkcert, /etc/resolver) is unix-only anyway. The in-memory mutex
// on ProcessManager still provides per-process safety, which is all this
// platform's binary realistically needs.
func (pm *ProcessManager) withFileLock(name string, fn func() error) error {
	_ = name
	return fn()
}
