package config

import "time"

// Config represents the tube configuration
type Config struct {
	Domain        string            `yaml:"domain" mapstructure:"domain"`
	TunnelPrefix  string            `yaml:"tunnel_prefix" mapstructure:"tunnel_prefix"`
	Directories   DirectoriesConfig `yaml:"directories" mapstructure:"directories"`
	Proxy         ProxyConfig       `yaml:"proxy" mapstructure:"proxy"`
	Nginx         NginxConfig       `yaml:"nginx" mapstructure:"nginx"`
	Dnsmasq       DnsmasqConfig     `yaml:"dnsmasq" mapstructure:"dnsmasq"`
	Tunnel        TunnelConfig      `yaml:"tunnel" mapstructure:"tunnel"`
	SSL           SSLConfig         `yaml:"ssl" mapstructure:"ssl"`
	Dashboard     DashboardConfig   `yaml:"dashboard" mapstructure:"dashboard"`
	Health        HealthConfig      `yaml:"health" mapstructure:"health"`
	Logging       LoggingConfig     `yaml:"logging" mapstructure:"logging"`
	Projects      map[string]int    `yaml:"projects" mapstructure:"projects"`
}

// DirectoriesConfig configures directory locations
type DirectoriesConfig struct {
	Config string `yaml:"config" mapstructure:"config"`
	Logs   string `yaml:"logs" mapstructure:"logs"`
	SSL    string `yaml:"ssl" mapstructure:"ssl"`
	PIDs   string `yaml:"pids" mapstructure:"pids"`
}

// ProxyConfig configures the proxy settings
type ProxyConfig struct {
	LocalDomain   string `yaml:"local_domain" mapstructure:"local_domain"`
	DashboardPort int    `yaml:"dashboard_port" mapstructure:"dashboard_port"`
	DocsPort      int    `yaml:"docs_port" mapstructure:"docs_port"`
}

// NginxConfig configures nginx
type NginxConfig struct {
	Binary    string `yaml:"binary" mapstructure:"binary"`
	HTTPPort  int    `yaml:"http_port" mapstructure:"http_port"`
	HTTPSPort int    `yaml:"https_port" mapstructure:"https_port"`
}

// DnsmasqConfig configures dnsmasq
type DnsmasqConfig struct {
	Binary string `yaml:"binary" mapstructure:"binary"`
	Port   int    `yaml:"port" mapstructure:"port"`
}

// TunnelConfig configures Cloudflare tunnel.
// Note: tunnel feature is not yet implemented; this exists only to round-trip
// config files that have the section.
type TunnelConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
	Binary  string `yaml:"binary" mapstructure:"binary"`
	Name    string `yaml:"name" mapstructure:"name"`
}

// SSLConfig configures SSL/TLS
type SSLConfig struct {
	Enabled      bool   `yaml:"enabled" mapstructure:"enabled"`
	AutoHTTPS    bool   `yaml:"auto_https" mapstructure:"auto_https"`
	MkcertBinary string `yaml:"mkcert_binary" mapstructure:"mkcert_binary"`
	CertFile     string `yaml:"cert_file" mapstructure:"cert_file"`
	KeyFile      string `yaml:"key_file" mapstructure:"key_file"`
	CAInstalled  bool   `yaml:"ca_installed" mapstructure:"ca_installed"`
}

// DashboardConfig configures the web dashboard
type DashboardConfig struct {
	Enabled bool       `yaml:"enabled" mapstructure:"enabled"`
	Port    int        `yaml:"port" mapstructure:"port"`
	Auth    AuthConfig `yaml:"auth" mapstructure:"auth"`
}

// AuthConfig configures dashboard authentication
type AuthConfig struct {
	Username     string `yaml:"username" mapstructure:"username"`
	PasswordHash string `yaml:"password_hash" mapstructure:"password_hash"`
}

// HealthConfig configures health monitoring
type HealthConfig struct {
	Enabled     bool          `yaml:"enabled" mapstructure:"enabled"`
	Interval    time.Duration `yaml:"interval" mapstructure:"interval"`
	AutoRestart bool          `yaml:"auto_restart" mapstructure:"auto_restart"`
}

// LoggingConfig configures logging
type LoggingConfig struct {
	Level  string `yaml:"level" mapstructure:"level"`
	Format string `yaml:"format" mapstructure:"format"`
}

// Project represents a single project configuration
type Project struct {
	Name   string `yaml:"name" mapstructure:"name"`
	Port   int    `yaml:"port" mapstructure:"port"`
	SSL    bool   `yaml:"ssl" mapstructure:"ssl"`
	Tunnel bool   `yaml:"tunnel" mapstructure:"tunnel"`
}
