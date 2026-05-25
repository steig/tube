package config

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// InteractiveInit guides the user through configuration setup
func InteractiveInit(configPath string) (*Config, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n🚀 Welcome to tube! Let's set up your configuration.")

	// Domain
	domain, err := promptForDomain(reader)
	if err != nil {
		return nil, err
	}

	// Tunnel prefix
	prefix, err := promptForPrefix(reader)
	if err != nil {
		return nil, err
	}

	// Local domain TLD
	localDomain, err := promptForLocalDomain(reader)
	if err != nil {
		return nil, err
	}

	// Always start from Defaults() so dashboard_port, ssl paths, directories,
	// etc. have sane values. The previous fallback zero-valued the struct,
	// which then failed Validate() on the next Load.
	cfg := Defaults()
	if existing, err := Load(configPath); err == nil {
		// Preserve any existing projects map across re-init.
		cfg.Projects = existing.Projects
		if cfg.Projects == nil {
			cfg.Projects = map[string]int{}
		}
	}

	cfg.Domain = domain
	cfg.TunnelPrefix = prefix
	cfg.Proxy.LocalDomain = localDomain

	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		return nil, err
	}

	// Save configuration
	if err := cfg.Save(configPath); err != nil {
		return nil, err
	}

	fmt.Printf("\n✓ Configuration saved to %s\n", configPath)

	// Setup SSL with mkcert
	if err := setupSSL(cfg, reader, configPath); err != nil {
		fmt.Printf("\n⚠ SSL setup skipped: %v\n", err)
		fmt.Println("  You can run 'tube ssl install' later to enable HTTPS")
	}

	fmt.Printf("\nYour setup:\n")
	fmt.Printf("  Domain: %s\n", cfg.Domain)
	fmt.Printf("  Tunnel prefix: %s\n", cfg.TunnelPrefix)
	fmt.Printf("  Local domain: %s\n", cfg.Proxy.LocalDomain)
	if cfg.SSL.Enabled && cfg.SSL.CertFile != "" {
		fmt.Printf("  HTTPS: enabled\n\n")
	} else {
		fmt.Printf("  HTTPS: disabled\n\n")
	}
	fmt.Printf("Next steps:\n")
	fmt.Printf("  1. Add your first project: tube add myapp 3000\n")
	fmt.Printf("  2. Start services: tube start\n")
	if cfg.SSL.Enabled {
		fmt.Printf("  3. Visit https://myapp%s in your browser\n\n", cfg.Proxy.LocalDomain)
	} else {
		fmt.Printf("  3. Visit http://myapp%s in your browser\n\n", cfg.Proxy.LocalDomain)
	}

	return cfg, nil
}

func promptForDomain(reader *bufio.Reader) (string, error) {
	fmt.Print("What's your domain name? (e.g., example.com): ")
	domain, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = "example.com"
	}

	// Basic validation
	if !isValidDomain(domain) {
		fmt.Printf("⚠ '%s' doesn't look like a valid domain. Using it anyway.\n", domain)
	}

	return domain, nil
}

func promptForPrefix(reader *bufio.Reader) (string, error) {
	fmt.Print("What prefix for tunnel subdomains? (e.g., dev-, local-): [dev-] ")
	prefix, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "dev-"
	}

	// Ensure it ends with hyphen
	if !strings.HasSuffix(prefix, "-") {
		prefix = prefix + "-"
	}

	return prefix, nil
}

func promptForLocalDomain(reader *bufio.Reader) (string, error) {
	fmt.Print("What local domain TLD? (e.g., .test, .local): [.test] ")
	domain, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = ".test"
	}

	// Ensure it starts with dot
	if !strings.HasPrefix(domain, ".") {
		domain = "." + domain
	}

	return domain, nil
}

func isValidDomain(domain string) bool {
	// Simple domain validation
	u, err := url.Parse("http://" + domain)
	if err != nil {
		return false
	}

	host := u.Host
	if host == "" {
		return false
	}

	// Check format: must have at least one dot
	if !strings.Contains(host, ".") {
		return false
	}

	// Check characters
	validDomain := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	return validDomain.MatchString(host)
}

// setupSSL configures SSL with mkcert
func setupSSL(cfg *Config, reader *bufio.Reader, configPath string) error {
	// Check if mkcert is installed
	mkcertPath, err := exec.LookPath(cfg.SSL.MkcertBinary)
	if err != nil {
		return fmt.Errorf("mkcert not installed (brew install mkcert)")
	}

	fmt.Println("\n🔒 Setting up HTTPS with mkcert...")

	// Check if CA is already installed
	homeDir, _ := os.UserHomeDir()
	caRoot := filepath.Join(homeDir, "Library", "Application Support", "mkcert")
	rootCA := filepath.Join(caRoot, "rootCA.pem")

	if _, err := os.Stat(rootCA); os.IsNotExist(err) {
		fmt.Println("Installing mkcert CA certificate (may require sudo)...")

		cmd := exec.Command(mkcertPath, "-install")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install CA: %w", err)
		}

		cfg.SSL.CAInstalled = true
	} else {
		fmt.Println("mkcert CA already installed")
		cfg.SSL.CAInstalled = true
	}

	// Generate wildcard certificate
	domain := strings.TrimPrefix(cfg.Proxy.LocalDomain, ".")
	certFile := filepath.Join(cfg.Directories.SSL, fmt.Sprintf("wildcard.%s.pem", domain))
	keyFile := filepath.Join(cfg.Directories.SSL, fmt.Sprintf("wildcard.%s-key.pem", domain))

	// Check if cert already exists
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		fmt.Printf("Generating wildcard certificate for *.%s...\n", domain)

		// Generate certificate
		wildcardDomain := fmt.Sprintf("*.%s", domain)
		cmd := exec.Command(mkcertPath, wildcardDomain, domain)
		cmd.Dir = cfg.Directories.SSL

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to generate certificate: %w", err)
		}

		// mkcert creates files like "_wildcard.test+1.pem" - rename them
		defaultCertName := filepath.Join(cfg.Directories.SSL, fmt.Sprintf("_wildcard.%s+1.pem", domain))
		defaultKeyName := filepath.Join(cfg.Directories.SSL, fmt.Sprintf("_wildcard.%s+1-key.pem", domain))

		if err := os.Rename(defaultCertName, certFile); err != nil {
			return fmt.Errorf("failed to rename certificate: %w", err)
		}
		if err := os.Rename(defaultKeyName, keyFile); err != nil {
			return fmt.Errorf("failed to rename key: %w", err)
		}
	} else {
		fmt.Println("Wildcard certificate already exists")
	}

	// Update config
	cfg.SSL.Enabled = true
	cfg.SSL.CertFile = certFile
	cfg.SSL.KeyFile = keyFile

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save SSL config: %w", err)
	}

	fmt.Println("✓ HTTPS configured successfully!")
	return nil
}
