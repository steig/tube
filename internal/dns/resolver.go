// Package dns provides DNS configuration management for macOS
package dns

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	resolverDir = "/etc/resolver"
)

// ResolverManager manages macOS /etc/resolver configuration
type ResolverManager struct {
	domain    string // e.g., "test"
	dnsServer string // e.g., "127.0.0.1"
}

// NewResolverManager creates a new ResolverManager
func NewResolverManager(domain, dnsServer string) *ResolverManager {
	// Strip leading dot if present
	domain = strings.TrimPrefix(domain, ".")

	return &ResolverManager{
		domain:    domain,
		dnsServer: dnsServer,
	}
}

// resolverPath returns the path to the resolver file
func (rm *ResolverManager) resolverPath() string {
	return filepath.Join(resolverDir, rm.domain)
}

// IsConfigured checks if the resolver is already configured
func (rm *ResolverManager) IsConfigured() (bool, error) {
	path := rm.resolverPath()

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read resolver file: %w", err)
	}

	// Check if it has our nameserver
	expected := fmt.Sprintf("nameserver %s", rm.dnsServer)
	return strings.Contains(string(content), expected), nil
}

// Setup configures the macOS resolver for the domain
// Requires sudo/root privileges
func (rm *ResolverManager) Setup() error {
	// Check if already configured
	configured, err := rm.IsConfigured()
	if err != nil {
		return err
	}
	if configured {
		return nil // Already done
	}

	// Create resolver directory if needed
	if err := os.MkdirAll(resolverDir, 0755); err != nil {
		return fmt.Errorf("failed to create resolver directory (need sudo?): %w", err)
	}

	// Write resolver file
	content := fmt.Sprintf("nameserver %s\n", rm.dnsServer)
	path := rm.resolverPath()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write resolver file (need sudo?): %w", err)
	}

	return nil
}

// Remove removes the resolver configuration
func (rm *ResolverManager) Remove() error {
	path := rm.resolverPath()

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove resolver file: %w", err)
	}

	return nil
}

// SetupWithSudo runs the setup with sudo privileges
func (rm *ResolverManager) SetupWithSudo() error {
	// Check if already configured
	configured, err := rm.IsConfigured()
	if err == nil && configured {
		return nil
	}

	// Create the resolver file content
	content := fmt.Sprintf("nameserver %s", rm.dnsServer)
	path := rm.resolverPath()

	// Use sudo to create directory and file
	// First ensure directory exists
	cmd := exec.Command("sudo", "mkdir", "-p", resolverDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create resolver directory: %w", err)
	}

	// Write the file using tee with sudo
	cmd = exec.Command("sudo", "tee", path)
	cmd.Stdin = strings.NewReader(content + "\n")
	cmd.Stdout = nil // Suppress output

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write resolver file: %w", err)
	}

	return nil
}

// RemoveWithSudo removes the resolver with sudo privileges
func (rm *ResolverManager) RemoveWithSudo() error {
	path := rm.resolverPath()

	cmd := exec.Command("sudo", "rm", "-f", path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove resolver file: %w", err)
	}

	return nil
}

// FlushDNSCache flushes the macOS DNS cache
func FlushDNSCache() error {
	cmd := exec.Command("sudo", "dscacheutil", "-flushcache")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to flush DNS cache: %w", err)
	}

	// killall -HUP mDNSResponder is best-effort; failures here aren't critical
	// because dscacheutil -flushcache above already did the meaningful work.
	_ = exec.Command("sudo", "killall", "-HUP", "mDNSResponder").Run()
	return nil
}

// Status returns the current resolver status
func (rm *ResolverManager) Status() (string, error) {
	configured, err := rm.IsConfigured()
	if err != nil {
		return "", err
	}

	if configured {
		return fmt.Sprintf("configured (*.%s -> %s)", rm.domain, rm.dnsServer), nil
	}

	return "not configured", nil
}
