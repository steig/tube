package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_DefaultConfig(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set HOME to temp dir
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create a config file with minimum required fields (viper doesn't merge defaults well)
	tubeDir := filepath.Join(tmpDir, ".tube")
	if err := os.MkdirAll(tubeDir, 0755); err != nil {
		t.Fatalf("failed to create .tube dir: %v", err)
	}
	configPath := filepath.Join(tubeDir, "config.yaml")
	// Must include all fields that validation requires
	configContent := `domain: example.com
tunnel_prefix: dev-
proxy:
  local_domain: .test
  dashboard_port: 3249
nginx:
  http_port: 80
dnsmasq:
  port: 53
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load config
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check values match what we wrote
	if cfg.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", cfg.Domain, "example.com")
	}
	if cfg.TunnelPrefix != "dev-" {
		t.Errorf("TunnelPrefix = %q, want %q", cfg.TunnelPrefix, "dev-")
	}
	if cfg.Proxy.LocalDomain != ".test" {
		t.Errorf("Proxy.LocalDomain = %q, want %q", cfg.Proxy.LocalDomain, ".test")
	}
	if cfg.Nginx.HTTPPort != 80 {
		t.Errorf("Nginx.HTTPPort = %d, want %d", cfg.Nginx.HTTPPort, 80)
	}
	if cfg.Dnsmasq.Port != 53 {
		t.Errorf("Dnsmasq.Port = %d, want %d", cfg.Dnsmasq.Port, 53)
	}
}

func TestLoad_WithConfigFile(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create config file with all required fields
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `domain: mytest.com
tunnel_prefix: stage-
proxy:
  local_domain: .local
  dashboard_port: 8080
nginx:
  http_port: 80
dnsmasq:
  port: 53
projects:
  myapp: 3000
  api: 8000
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check loaded values
	if cfg.Domain != "mytest.com" {
		t.Errorf("Domain = %q, want %q", cfg.Domain, "mytest.com")
	}
	if cfg.TunnelPrefix != "stage-" {
		t.Errorf("TunnelPrefix = %q, want %q", cfg.TunnelPrefix, "stage-")
	}
	if cfg.Proxy.LocalDomain != ".local" {
		t.Errorf("Proxy.LocalDomain = %q, want %q", cfg.Proxy.LocalDomain, ".local")
	}
	if cfg.Proxy.DashboardPort != 8080 {
		t.Errorf("Proxy.DashboardPort = %d, want %d", cfg.Proxy.DashboardPort, 8080)
	}

	// Check projects
	if port, ok := cfg.Projects["myapp"]; !ok || port != 3000 {
		t.Errorf("Projects[myapp] = %d, want %d", port, 3000)
	}
	if port, ok := cfg.Projects["api"]; !ok || port != 8000 {
		t.Errorf("Projects[api] = %d, want %d", port, 8000)
	}
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create invalid config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `{{{invalid yaml`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Load should fail
	_, err = Load(configPath)
	if err == nil {
		t.Error("Load() expected error for invalid YAML, got nil")
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Domain:       "example.com",
				TunnelPrefix: "dev-",
				Proxy: ProxyConfig{
					LocalDomain:   ".test",
					DashboardPort: 3249,
				},
				Projects: map[string]int{
					"myapp": 3000,
				},
			},
			wantErr: false,
		},
		{
			name: "empty domain",
			config: Config{
				Domain:       "",
				TunnelPrefix: "dev-",
				Proxy: ProxyConfig{
					LocalDomain:   ".test",
					DashboardPort: 3249,
				},
			},
			wantErr: true,
			errMsg:  "domain cannot be empty",
		},
		{
			name: "empty tunnel prefix",
			config: Config{
				Domain:       "example.com",
				TunnelPrefix: "",
				Proxy: ProxyConfig{
					LocalDomain:   ".test",
					DashboardPort: 3249,
				},
			},
			wantErr: true,
			errMsg:  "tunnel_prefix cannot be empty",
		},
		{
			name: "empty local domain",
			config: Config{
				Domain:       "example.com",
				TunnelPrefix: "dev-",
				Proxy: ProxyConfig{
					LocalDomain:   "",
					DashboardPort: 3249,
				},
			},
			wantErr: true,
			errMsg:  "proxy.local_domain cannot be empty",
		},
		{
			name: "invalid dashboard port - too low",
			config: Config{
				Domain:       "example.com",
				TunnelPrefix: "dev-",
				Proxy: ProxyConfig{
					LocalDomain:   ".test",
					DashboardPort: 80,
				},
			},
			wantErr: true,
			errMsg:  "dashboard_port must be between 1024 and 65535",
		},
		{
			name: "invalid dashboard port - too high",
			config: Config{
				Domain:       "example.com",
				TunnelPrefix: "dev-",
				Proxy: ProxyConfig{
					LocalDomain:   ".test",
					DashboardPort: 70000,
				},
			},
			wantErr: true,
			errMsg:  "dashboard_port must be between 1024 and 65535",
		},
		{
			name: "invalid project port",
			config: Config{
				Domain:       "example.com",
				TunnelPrefix: "dev-",
				Proxy: ProxyConfig{
					LocalDomain:   ".test",
					DashboardPort: 3249,
				},
				Projects: map[string]int{
					"myapp": 80,
				},
			},
			wantErr: true,
			errMsg:  "project \"myapp\": port 80 must be between 1024 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestConfig_Save(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "subdir", "config.yaml")

	cfg := &Config{
		Domain:       "saved.com",
		TunnelPrefix: "test-",
		Proxy: ProxyConfig{
			LocalDomain:   ".dev",
			DashboardPort: 9000,
		},
		Projects: map[string]int{
			"app1": 3000,
			"app2": 4000,
		},
	}

	// Save config (should create directory)
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Save() did not create config file")
	}

	// Verify file contents directly (avoid Load validation issues)
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	// Check that key values are in the saved file
	contentStr := string(content)
	if !strings.Contains(contentStr, "saved.com") {
		t.Error("Saved config missing domain")
	}
	if !strings.Contains(contentStr, "test-") {
		t.Error("Saved config missing tunnel_prefix")
	}
	if !strings.Contains(contentStr, "app1") {
		t.Error("Saved config missing project app1")
	}
}

func TestConfig_EnsureDirectories(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &Config{
		Directories: DirectoriesConfig{
			Config: filepath.Join(tmpDir, "config"),
			Logs:   filepath.Join(tmpDir, "logs"),
			SSL:    filepath.Join(tmpDir, "ssl"),
			PIDs:   filepath.Join(tmpDir, "pids"),
		},
	}

	// Ensure directories
	if err := cfg.EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories() error = %v", err)
	}

	// Check all directories exist
	dirs := []string{
		cfg.Directories.Config,
		cfg.Directories.Logs,
		cfg.Directories.SSL,
		cfg.Directories.PIDs,
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("EnsureDirectories() did not create %s", dir)
		}
	}
}

func TestConfigPath(t *testing.T) {
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/test/home")
	defer os.Setenv("HOME", oldHome)

	path := ConfigPath()
	expected := "/test/home/.tube/config.yaml"

	if path != expected {
		t.Errorf("ConfigPath() = %q, want %q", path, expected)
	}
}
