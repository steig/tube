package proxy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/service"
)

// setupTestEnv creates a test environment with templates and returns cleanup function
func setupTestEnv(t *testing.T) (tmpDir string, cfg *config.Config, pm *service.ProcessManager, cleanup func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create template directories
	nginxTmplDir := filepath.Join(tmpDir, "templates", "nginx")
	dnsmasqTmplDir := filepath.Join(tmpDir, "templates", "dnsmasq")
	if err := os.MkdirAll(nginxTmplDir, 0755); err != nil {
		t.Fatalf("failed to create nginx template dir: %v", err)
	}
	if err := os.MkdirAll(dnsmasqTmplDir, 0755); err != nil {
		t.Fatalf("failed to create dnsmasq template dir: %v", err)
	}

	// Create minimal test templates
	mainTmpl := `# Test nginx config
worker_processes 1;
http {
    {{range $name, $port := .Projects}}
    upstream {{$name}} {
        server 127.0.0.1:{{$port}};
    }
    {{end}}
}
`
	if err := os.WriteFile(filepath.Join(nginxTmplDir, "main.conf.tmpl"), []byte(mainTmpl), 0644); err != nil {
		t.Fatalf("failed to write main template: %v", err)
	}

	projTmpl := `# Generated at {{.GeneratedAt}}
{{range $name, $port := .Projects}}
# Project: {{$name}} -> {{$port}}
{{end}}
`
	if err := os.WriteFile(filepath.Join(nginxTmplDir, "projects.conf.tmpl"), []byte(projTmpl), 0644); err != nil {
		t.Fatalf("failed to write projects template: %v", err)
	}

	dnsmasqTmpl := `# Dnsmasq test config
listen-address=127.0.0.1
address=/{{.Proxy.LocalDomain}}/127.0.0.1
{{range $name, $port := .Projects}}
address=/{{$name}}.{{$.Proxy.LocalDomain}}/127.0.0.1
{{end}}
`
	if err := os.WriteFile(filepath.Join(dnsmasqTmplDir, "dnsmasq.conf.tmpl"), []byte(dnsmasqTmpl), 0644); err != nil {
		t.Fatalf("failed to write dnsmasq template: %v", err)
	}

	// Create config
	cfg = &config.Config{
		Domain:       "example.com",
		TunnelPrefix: "dev-",
		Directories: config.DirectoriesConfig{
			Config: filepath.Join(tmpDir, "config"),
			Logs:   filepath.Join(tmpDir, "logs"),
			PIDs:   filepath.Join(tmpDir, "pids"),
		},
		Proxy: config.ProxyConfig{
			LocalDomain:   "test",
			DashboardPort: 3249,
		},
		Nginx: config.NginxConfig{
			Binary:    "nginx",
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Projects: map[string]int{
			"myapp": 3000,
			"api":   8080,
		},
	}

	// Create ProcessManager
	pm, err = service.NewProcessManager(cfg.Directories.PIDs)
	if err != nil {
		t.Fatalf("failed to create process manager: %v", err)
	}

	// Change to temp dir so templates are found
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)

	cleanup = func() {
		os.Chdir(oldWd)
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cfg, pm, cleanup
}

func TestNewNginxManager(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	nm, err := NewNginxManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewNginxManager() error = %v", err)
	}

	if nm == nil {
		t.Fatal("NewNginxManager() returned nil")
	}

	// Verify config directory was created
	if _, err := os.Stat(cfg.Directories.Config); os.IsNotExist(err) {
		t.Error("NewNginxManager() did not create config directory")
	}
}

func TestNewDnsmasqManager(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	dm, err := NewDnsmasqManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewDnsmasqManager() error = %v", err)
	}

	if dm == nil {
		t.Fatal("NewDnsmasqManager() returned nil")
	}
}

func TestNginxManager_GenerateConfig(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	nm, err := NewNginxManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewNginxManager() error = %v", err)
	}

	config, err := nm.GenerateConfig()
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	// Verify config contains expected content
	if !strings.Contains(config, "worker_processes") {
		t.Error("GenerateConfig() missing worker_processes directive")
	}

	// Verify projects are included
	if !strings.Contains(config, "upstream myapp") {
		t.Error("GenerateConfig() missing myapp upstream")
	}
	if !strings.Contains(config, "upstream api") {
		t.Error("GenerateConfig() missing api upstream")
	}
	if !strings.Contains(config, "127.0.0.1:3000") {
		t.Error("GenerateConfig() missing myapp port")
	}
	if !strings.Contains(config, "127.0.0.1:8080") {
		t.Error("GenerateConfig() missing api port")
	}
}

func TestNginxManager_GenerateProjectsConfig(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	nm, err := NewNginxManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewNginxManager() error = %v", err)
	}

	config, err := nm.GenerateProjectsConfig()
	if err != nil {
		t.Fatalf("GenerateProjectsConfig() error = %v", err)
	}

	// Verify generated timestamp is included
	if !strings.Contains(config, "Generated at") {
		t.Error("GenerateProjectsConfig() missing timestamp")
	}

	// Verify projects are included
	if !strings.Contains(config, "myapp") {
		t.Error("GenerateProjectsConfig() missing myapp")
	}
	if !strings.Contains(config, "api") {
		t.Error("GenerateProjectsConfig() missing api")
	}
}

func TestNginxManager_WriteConfig(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	nm, err := NewNginxManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewNginxManager() error = %v", err)
	}

	if err := nm.WriteConfig(); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	// Verify files were created
	mainPath := filepath.Join(cfg.Directories.Config, "nginx.conf")
	if _, err := os.Stat(mainPath); os.IsNotExist(err) {
		t.Error("WriteConfig() did not create nginx.conf")
	}

	projPath := filepath.Join(cfg.Directories.Config, "projects.conf")
	if _, err := os.Stat(projPath); os.IsNotExist(err) {
		t.Error("WriteConfig() did not create projects.conf")
	}

	// Verify content
	mainContent, _ := os.ReadFile(mainPath)
	if !strings.Contains(string(mainContent), "worker_processes") {
		t.Error("nginx.conf missing expected content")
	}
}

func TestDnsmasqManager_GenerateConfig(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	dm, err := NewDnsmasqManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewDnsmasqManager() error = %v", err)
	}

	config, err := dm.GenerateConfig()
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	// Verify config contains expected content
	if !strings.Contains(config, "listen-address=127.0.0.1") {
		t.Error("GenerateConfig() missing listen-address")
	}
	if !strings.Contains(config, "address=/test/127.0.0.1") {
		t.Error("GenerateConfig() missing local domain address")
	}

	// Verify projects are included
	if !strings.Contains(config, "myapp.test") {
		t.Error("GenerateConfig() missing myapp.test")
	}
	if !strings.Contains(config, "api.test") {
		t.Error("GenerateConfig() missing api.test")
	}
}

func TestDnsmasqManager_WriteConfig(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	dm, err := NewDnsmasqManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewDnsmasqManager() error = %v", err)
	}

	if err := dm.WriteConfig(); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	// Verify file was created
	confPath := filepath.Join(cfg.Directories.Config, "dnsmasq.conf")
	if _, err := os.Stat(confPath); os.IsNotExist(err) {
		t.Error("WriteConfig() did not create dnsmasq.conf")
	}

	// Verify content
	content, _ := os.ReadFile(confPath)
	if !strings.Contains(string(content), "listen-address") {
		t.Error("dnsmasq.conf missing expected content")
	}
}

func TestNginxManager_EmptyProjects(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	// Clear projects
	cfg.Projects = map[string]int{}

	nm, err := NewNginxManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewNginxManager() error = %v", err)
	}

	config, err := nm.GenerateConfig()
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	// Should still generate valid config
	if !strings.Contains(config, "worker_processes") {
		t.Error("GenerateConfig() with no projects missing worker_processes")
	}

	// Should not have upstream blocks
	if strings.Contains(config, "upstream") {
		t.Error("GenerateConfig() with no projects should not have upstream blocks")
	}
}

func TestDnsmasqManager_EmptyProjects(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	// Clear projects
	cfg.Projects = map[string]int{}

	dm, err := NewDnsmasqManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewDnsmasqManager() error = %v", err)
	}

	config, err := dm.GenerateConfig()
	if err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}

	// Should still have base config
	if !strings.Contains(config, "listen-address") {
		t.Error("GenerateConfig() with no projects missing listen-address")
	}

	// Should still have main domain
	if !strings.Contains(config, "address=/test/") {
		t.Error("GenerateConfig() with no projects missing local domain")
	}
}

func TestNginxManager_IsRunning_NotStarted(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	nm, err := NewNginxManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewNginxManager() error = %v", err)
	}

	isRunning, err := nm.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() error = %v", err)
	}
	if isRunning {
		t.Error("IsRunning() = true, want false for not-started nginx")
	}
}

func TestDnsmasqManager_IsRunning_NotStarted(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	dm, err := NewDnsmasqManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewDnsmasqManager() error = %v", err)
	}

	isRunning, err := dm.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() error = %v", err)
	}
	if isRunning {
		t.Error("IsRunning() = true, want false for not-started dnsmasq")
	}
}

func TestNginxManager_Status_NotStarted(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	nm, err := NewNginxManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewNginxManager() error = %v", err)
	}

	status, err := nm.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != "stopped" {
		t.Errorf("Status() = %q, want %q", status, "stopped")
	}
}

func TestDnsmasqManager_Status_NotStarted(t *testing.T) {
	_, cfg, pm, cleanup := setupTestEnv(t)
	defer cleanup()

	dm, err := NewDnsmasqManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewDnsmasqManager() error = %v", err)
	}

	status, err := dm.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != "stopped" {
		t.Errorf("Status() = %q, want %q", status, "stopped")
	}
}
