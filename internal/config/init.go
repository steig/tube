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

// InteractiveInit guides the user through configuration setup.
// Only prompts for things that affect runtime behavior today — the
// tunnel-related fields (Domain, TunnelPrefix) use defaults silently since
// the Cloudflare Tunnel feature isn't shipping yet. Re-enable those prompts
// by passing withTunnel=true.
func InteractiveInit(configPath string, withTunnel bool) (*Config, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nSetting up tube.")

	// Local TLD is the only prompt that affects runtime today.
	localDomain, err := promptForLocalDomain(reader)
	if err != nil {
		return nil, err
	}

	// Optional tunnel-related prompts (planned feature; off by default).
	var (
		domain = ""
		prefix = ""
	)
	if withTunnel {
		domain, err = promptForDomain(reader)
		if err != nil {
			return nil, err
		}
		prefix, err = promptForPrefix(reader)
		if err != nil {
			return nil, err
		}
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

	if domain != "" {
		cfg.Domain = domain
	}
	if prefix != "" {
		cfg.TunnelPrefix = prefix
	}
	cfg.Proxy.LocalDomain = localDomain

	if err := cfg.EnsureDirectories(); err != nil {
		return nil, err
	}
	if err := cfg.Save(configPath); err != nil {
		return nil, err
	}

	fmt.Printf("\n✓ Configuration saved to %s\n", configPath)

	// Setup SSL with mkcert
	if err := setupSSL(cfg, reader, configPath); err != nil {
		fmt.Printf("\n⚠ SSL setup skipped: %v\n", err)
		fmt.Println("  Run 'tube ssl install' later to enable HTTPS")
	}

	fmt.Printf("\nYour setup:\n")
	fmt.Printf("  Local domain: %s\n", cfg.Proxy.LocalDomain)
	if cfg.SSL.Enabled && cfg.SSL.CertFile != "" {
		fmt.Printf("  HTTPS:        enabled\n")
	} else {
		fmt.Printf("  HTTPS:        disabled\n")
	}

	scheme := "http"
	if cfg.SSL.Enabled {
		scheme = "https"
	}
	tld := strings.TrimPrefix(cfg.Proxy.LocalDomain, ".")

	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Wire up macOS DNS (one-time, needs sudo): sudo tube setup")
	fmt.Println("  2. Add your first project:                   tube add myapp 3000")
	fmt.Println("  3. Start services:                           tube start")
	fmt.Printf("  4. Open it in a browser:                     %s://myapp.%s\n\n", scheme, tld)

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

	if !strings.HasSuffix(prefix, "-") {
		prefix = prefix + "-"
	}

	return prefix, nil
}

func promptForLocalDomain(reader *bufio.Reader) (string, error) {
	fmt.Print("Local domain TLD? (e.g., .test, .local) [.test]: ")
	domain, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = ".test"
	}

	if !strings.HasPrefix(domain, ".") {
		domain = "." + domain
	}

	return domain, nil
}

func isValidDomain(domain string) bool {
	u, err := url.Parse("http://" + domain)
	if err != nil {
		return false
	}

	host := u.Host
	if host == "" {
		return false
	}

	if !strings.Contains(host, ".") {
		return false
	}

	validDomain := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	return validDomain.MatchString(host)
}

// setupSSL configures SSL with mkcert
func setupSSL(cfg *Config, _ *bufio.Reader, configPath string) error {
	mkcertPath, err := exec.LookPath(cfg.SSL.MkcertBinary)
	if err != nil {
		return fmt.Errorf("mkcert not installed (brew install mkcert)")
	}

	fmt.Println("\nSetting up HTTPS with mkcert...")

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

	domain := strings.TrimPrefix(cfg.Proxy.LocalDomain, ".")
	certFile := filepath.Join(cfg.Directories.SSL, fmt.Sprintf("wildcard.%s.pem", domain))
	keyFile := filepath.Join(cfg.Directories.SSL, fmt.Sprintf("wildcard.%s-key.pem", domain))

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		fmt.Printf("Generating wildcard certificate for *.%s...\n", domain)
		cmd := exec.Command(mkcertPath,
			"-cert-file", certFile,
			"-key-file", keyFile,
			fmt.Sprintf("*.%s", domain),
			domain,
		)
		cmd.Dir = cfg.Directories.SSL
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to generate certificate: %w", err)
		}
	} else {
		fmt.Println("Wildcard certificate already exists")
	}

	cfg.SSL.Enabled = true
	cfg.SSL.CertFile = certFile
	cfg.SSL.KeyFile = keyFile

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save SSL config: %w", err)
	}

	fmt.Println("✓ HTTPS configured")
	return nil
}
