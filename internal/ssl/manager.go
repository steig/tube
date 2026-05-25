package ssl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/steig/tube/internal/config"
)

// CertManager manages SSL certificates using mkcert
type CertManager struct {
	config     *config.Config
	mkcertPath string
	certsDir   string
}

// CertInfo holds information about a certificate.
// The struct itself describes a cert that exists on disk. Callers check
// existence via CertManager.CertExists rather than a redundant field.
type CertInfo struct {
	Domain   string
	CertFile string
	KeyFile  string
}

// NewCertManager creates a new CertManager
func NewCertManager(cfg *config.Config) (*CertManager, error) {
	// Find mkcert binary
	mkcertPath, err := exec.LookPath(cfg.SSL.MkcertBinary)
	if err != nil {
		return nil, fmt.Errorf("mkcert not found: %w (install with: brew install mkcert)", err)
	}

	// Ensure SSL directory exists
	certsDir := cfg.Directories.SSL
	if err := os.MkdirAll(certsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create SSL directory: %w", err)
	}

	return &CertManager{
		config:     cfg,
		mkcertPath: mkcertPath,
		certsDir:   certsDir,
	}, nil
}

// InstallCA installs the mkcert CA to the system trust store
// This requires sudo/admin privileges
func (cm *CertManager) InstallCA() error {
	cmd := exec.Command(cm.mkcertPath, "-install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install CA: %w", err)
	}

	return nil
}

// IsCAInstalled checks if the mkcert CA is installed
func (cm *CertManager) IsCAInstalled() bool {
	// Check CAROOT environment or default location
	caRoot := os.Getenv("CAROOT")
	if caRoot == "" {
		// Default mkcert CA location on macOS
		homeDir, _ := os.UserHomeDir()
		caRoot = filepath.Join(homeDir, "Library", "Application Support", "mkcert")
	}

	rootCA := filepath.Join(caRoot, "rootCA.pem")
	_, err := os.Stat(rootCA)
	return err == nil
}

// GenerateWildcard generates a wildcard certificate for the given domain.
// E.g. GenerateWildcard("test") produces certs for *.test and test.
//
// The cleanest approach is to pass mkcert -cert-file / -key-file flags so we
// never have to guess the filename it would have used. Older versions of
// mkcert support these flags (they were added long ago).
func (cm *CertManager) GenerateWildcard(domain string) (*CertInfo, error) {
	domain = strings.TrimPrefix(domain, ".")

	certFile := filepath.Join(cm.certsDir, fmt.Sprintf("wildcard.%s.pem", domain))
	keyFile := filepath.Join(cm.certsDir, fmt.Sprintf("wildcard.%s-key.pem", domain))

	cmd := exec.Command(cm.mkcertPath,
		"-cert-file", certFile,
		"-key-file", keyFile,
		fmt.Sprintf("*.%s", domain),
		domain,
	)
	cmd.Dir = cm.certsDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %w\n%s", err, stderr.String())
	}

	return &CertInfo{Domain: domain, CertFile: certFile, KeyFile: keyFile}, nil
}

// GenerateCert generates a certificate for specific domains
func (cm *CertManager) GenerateCert(name string, domains ...string) (*CertInfo, error) {
	if len(domains) == 0 {
		return nil, fmt.Errorf("at least one domain is required")
	}

	// Certificate file names
	certFile := filepath.Join(cm.certsDir, fmt.Sprintf("%s.pem", name))
	keyFile := filepath.Join(cm.certsDir, fmt.Sprintf("%s-key.pem", name))

	// Build mkcert command
	args := append([]string{"-cert-file", certFile, "-key-file", keyFile}, domains...)
	cmd := exec.Command(cm.mkcertPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %w\n%s", err, stderr.String())
	}

	return &CertInfo{Domain: domains[0], CertFile: certFile, KeyFile: keyFile}, nil
}

// GetCertPaths returns the certificate and key paths for the given domain
func (cm *CertManager) GetCertPaths(domain string) (certFile, keyFile string) {
	domain = strings.TrimPrefix(domain, ".")
	certFile = filepath.Join(cm.certsDir, fmt.Sprintf("wildcard.%s.pem", domain))
	keyFile = filepath.Join(cm.certsDir, fmt.Sprintf("wildcard.%s-key.pem", domain))
	return
}

// CertExists checks if a certificate exists for the given domain
func (cm *CertManager) CertExists(domain string) bool {
	certFile, keyFile := cm.GetCertPaths(domain)

	_, certErr := os.Stat(certFile)
	_, keyErr := os.Stat(keyFile)

	return certErr == nil && keyErr == nil
}

// GetCertInfo returns information about a certificate
func (cm *CertManager) GetCertInfo(domain string) *CertInfo {
	domain = strings.TrimPrefix(domain, ".")
	certFile, keyFile := cm.GetCertPaths(domain)
	return &CertInfo{Domain: domain, CertFile: certFile, KeyFile: keyFile}
}

// ListCerts lists all certificates in the SSL directory
func (cm *CertManager) ListCerts() ([]CertInfo, error) {
	entries, err := os.ReadDir(cm.certsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSL directory: %w", err)
	}

	// Find .pem files (excluding -key.pem). Only return certs whose matching
	// key file is also present — orphaned certs aren't usable.
	var certs []CertInfo
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, "-key.pem") {
			continue
		}
		domain := strings.TrimSuffix(name, ".pem")
		domain = strings.TrimPrefix(domain, "wildcard.")

		certFile := filepath.Join(cm.certsDir, name)
		keyFile := filepath.Join(cm.certsDir, strings.TrimSuffix(name, ".pem")+"-key.pem")
		if _, err := os.Stat(keyFile); err != nil {
			continue
		}
		certs = append(certs, CertInfo{Domain: domain, CertFile: certFile, KeyFile: keyFile})
	}
	return certs, nil
}

// EnsureWildcardCert ensures a wildcard certificate exists for the local domain
// If it doesn't exist, it generates one
func (cm *CertManager) EnsureWildcardCert() (*CertInfo, error) {
	domain := strings.TrimPrefix(cm.config.Proxy.LocalDomain, ".")

	if cm.CertExists(domain) {
		return cm.GetCertInfo(domain), nil
	}

	return cm.GenerateWildcard(domain)
}

// Status returns the current SSL status
type Status struct {
	Enabled      bool
	CAInstalled  bool
	CertExists   bool
	CertFile     string
	KeyFile      string
	LocalDomain  string
	MkcertPath   string
}

// GetStatus returns the current SSL status
func (cm *CertManager) GetStatus() *Status {
	domain := strings.TrimPrefix(cm.config.Proxy.LocalDomain, ".")
	certFile, keyFile := cm.GetCertPaths(domain)

	return &Status{
		Enabled:      cm.config.SSL.Enabled,
		CAInstalled:  cm.IsCAInstalled(),
		CertExists:   cm.CertExists(domain),
		CertFile:     certFile,
		KeyFile:      keyFile,
		LocalDomain:  domain,
		MkcertPath:   cm.mkcertPath,
	}
}
