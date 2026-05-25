package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

const (
	configFileName = "config"
	configFileType = "yaml"
)

// Load loads the tube configuration
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file settings
	v.SetConfigName(configFileName)
	v.SetConfigType(configFileType)

	// Use provided path or default
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		configDir := filepath.Join(os.Getenv("HOME"), ".tube")
		v.AddConfigPath(configDir)
	}

	// Set defaults
	setDefaults(v)

	// Environment variables with TUBE_ prefix
	v.SetEnvPrefix("TUBE")
	v.AutomaticEnv()

	// Read config file (ignore if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	// Unmarshal to struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Defaults returns a Config populated with the same defaults Load applies.
// Use this when you need a fresh defaults-only config without touching disk
// (e.g. interactive init when no config file exists yet).
func Defaults() *Config {
	v := viper.New()
	setDefaults(v)
	var cfg Config
	_ = v.Unmarshal(&cfg)
	if cfg.Projects == nil {
		cfg.Projects = map[string]int{}
	}
	return &cfg
}

// setDefaults sets default values for the configuration
func setDefaults(v *viper.Viper) {
	homeDir := os.Getenv("HOME")
	configDir := filepath.Join(homeDir, ".tube")

	// Domain and tunnel settings
	v.SetDefault("domain", "example.com")
	v.SetDefault("tunnel_prefix", "dev-")

	// Directories
	v.SetDefault("directories.config", configDir)
	v.SetDefault("directories.logs", filepath.Join(configDir, "logs"))
	v.SetDefault("directories.ssl", filepath.Join(configDir, "ssl"))
	v.SetDefault("directories.pids", filepath.Join(configDir, "pids"))

	// Proxy settings
	v.SetDefault("proxy.local_domain", ".test")
	v.SetDefault("proxy.dashboard_port", 3249)
	v.SetDefault("proxy.docs_port", 3250)

	// nginx settings
	v.SetDefault("nginx.binary", "nginx")
	v.SetDefault("nginx.http_port", 80)
	v.SetDefault("nginx.https_port", 443)

	// dnsmasq settings
	v.SetDefault("dnsmasq.binary", "dnsmasq")
	v.SetDefault("dnsmasq.port", 53)

	// Tunnel settings
	v.SetDefault("tunnel.enabled", false)
	v.SetDefault("tunnel.binary", "cloudflared")
	v.SetDefault("tunnel.name", "tube-tunnel")

	// SSL settings
	v.SetDefault("ssl.enabled", true)
	v.SetDefault("ssl.auto_https", true)
	v.SetDefault("ssl.mkcert_binary", "mkcert")
	v.SetDefault("ssl.cert_file", filepath.Join(configDir, "ssl", "wildcard.test.pem"))
	v.SetDefault("ssl.key_file", filepath.Join(configDir, "ssl", "wildcard.test-key.pem"))
	v.SetDefault("ssl.ca_installed", false)

	// Dashboard settings
	v.SetDefault("dashboard.enabled", true)
	v.SetDefault("dashboard.port", 3249)
	v.SetDefault("dashboard.auth.username", "admin")

	// Health settings
	v.SetDefault("health.enabled", true)
	v.SetDefault("health.interval", 30*time.Second)
	v.SetDefault("health.auto_restart", true)

	// Logging settings
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Projects (empty by default)
	v.SetDefault("projects", map[string]int{})
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	if c.TunnelPrefix == "" {
		return fmt.Errorf("tunnel_prefix cannot be empty")
	}

	if c.Proxy.LocalDomain == "" {
		return fmt.Errorf("proxy.local_domain cannot be empty")
	}

	if c.Proxy.DashboardPort < 1024 || c.Proxy.DashboardPort > 65535 {
		return fmt.Errorf("dashboard_port must be between 1024 and 65535")
	}

	// Validate project ports
	for name, port := range c.Projects {
		if port < 1024 || port > 65535 {
			return fmt.Errorf("project %q: port %d must be between 1024 and 65535", name, port)
		}
	}

	return nil
}

// Save saves the configuration to file
func (c *Config) Save(configPath string) error {
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigType(configFileType)

	// Manually set values instead of merging struct
	v.Set("domain", c.Domain)
	v.Set("tunnel_prefix", c.TunnelPrefix)
	v.Set("directories", c.Directories)
	v.Set("proxy", c.Proxy)
	v.Set("nginx", c.Nginx)
	v.Set("dnsmasq", c.Dnsmasq)
	v.Set("tunnel", c.Tunnel)
	v.Set("ssl", c.SSL)
	v.Set("dashboard", c.Dashboard)
	v.Set("health", c.Health)
	v.Set("logging", c.Logging)
	v.Set("projects", c.Projects)

	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".tube", "config.yaml")
}

// EnsureDirectories creates required directories
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.Directories.Config,
		c.Directories.Logs,
		c.Directories.SSL,
		c.Directories.PIDs,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
