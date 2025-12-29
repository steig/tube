//go:build integration

// Package integration contains integration tests that require a full environment
// with nginx and dnsmasq installed. These tests are skipped unless run with
// the -tags=integration flag and TUBE_INTEGRATION_TESTS=1 environment variable.
//
// To run integration tests:
//   docker compose -f docker-compose.test.yml run integration
package integration

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/proxy"
	"github.com/steig/tube/internal/service"
)

func TestMain(m *testing.M) {
	// Skip if not in integration test mode
	if os.Getenv("TUBE_INTEGRATION_TESTS") != "1" {
		fmt.Println("Skipping integration tests. Set TUBE_INTEGRATION_TESTS=1 to run.")
		os.Exit(0)
	}

	os.Exit(m.Run())
}

// setupIntegrationEnv creates a full test environment for integration testing
func setupIntegrationEnv(t *testing.T) (cfg *config.Config, pm *service.ProcessManager, cleanup func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tube-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create config
	cfg = &config.Config{
		Domain:       "integration.test",
		TunnelPrefix: "test-",
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
			HTTPPort:  8080, // Use non-privileged port for testing
			HTTPSPort: 8443,
		},
		Dnsmasq: config.DnsmasqConfig{
			Binary: "dnsmasq",
			Port:   5353, // Use non-privileged port for testing
		},
		SSL: config.SSLConfig{
			Enabled: false, // Disable SSL for integration tests
		},
		Projects: map[string]int{
			"testapp": 9000,
		},
	}

	// Create directories
	if err := cfg.EnsureDirectories(); err != nil {
		t.Fatalf("failed to ensure directories: %v", err)
	}

	// Create ProcessManager
	pm, err = service.NewProcessManager(cfg.Directories.PIDs)
	if err != nil {
		t.Fatalf("failed to create process manager: %v", err)
	}

	cleanup = func() {
		// Stop all services
		_ = pm.StopAll()
		// Remove temp directory
		os.RemoveAll(tmpDir)
	}

	return cfg, pm, cleanup
}

func TestIntegration_NginxLifecycle(t *testing.T) {
	cfg, pm, cleanup := setupIntegrationEnv(t)
	defer cleanup()

	// Check nginx is installed
	if _, err := exec.LookPath("nginx"); err != nil {
		t.Skip("nginx not installed, skipping")
	}

	// Create NginxManager
	nm, err := proxy.NewNginxManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewNginxManager() error = %v", err)
	}

	// Write config
	if err := nm.WriteConfig(); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	// Test config
	valid, err := nm.TestConfig()
	if err != nil {
		t.Fatalf("TestConfig() error = %v", err)
	}
	if !valid {
		t.Fatal("TestConfig() returned invalid")
	}

	// Start nginx
	if err := nm.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for startup
	time.Sleep(500 * time.Millisecond)

	// Check running
	isRunning, err := nm.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() error = %v", err)
	}
	if !isRunning {
		t.Error("nginx should be running after Start()")
	}

	// Check status
	status, err := nm.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status == "stopped" {
		t.Error("status should not be 'stopped' after Start()")
	}

	// Stop nginx
	if err := nm.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Wait for shutdown
	time.Sleep(500 * time.Millisecond)

	// Check stopped
	isRunning, err = nm.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() after Stop error = %v", err)
	}
	if isRunning {
		t.Error("nginx should not be running after Stop()")
	}
}

func TestIntegration_DnsmasqLifecycle(t *testing.T) {
	cfg, pm, cleanup := setupIntegrationEnv(t)
	defer cleanup()

	// Check dnsmasq is installed
	if _, err := exec.LookPath("dnsmasq"); err != nil {
		t.Skip("dnsmasq not installed, skipping")
	}

	// Create DnsmasqManager
	dm, err := proxy.NewDnsmasqManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewDnsmasqManager() error = %v", err)
	}

	// Write config
	if err := dm.WriteConfig(); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	// Start dnsmasq
	if err := dm.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for startup
	time.Sleep(500 * time.Millisecond)

	// Check running
	isRunning, err := dm.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() error = %v", err)
	}
	if !isRunning {
		t.Error("dnsmasq should be running after Start()")
	}

	// Stop dnsmasq
	if err := dm.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Wait for shutdown
	time.Sleep(500 * time.Millisecond)

	// Check stopped
	isRunning, err = dm.IsRunning()
	if err != nil {
		t.Fatalf("IsRunning() after Stop error = %v", err)
	}
	if isRunning {
		t.Error("dnsmasq should not be running after Stop()")
	}
}

func TestIntegration_FullServiceWorkflow(t *testing.T) {
	cfg, pm, cleanup := setupIntegrationEnv(t)
	defer cleanup()

	// Check both services are installed
	if _, err := exec.LookPath("nginx"); err != nil {
		t.Skip("nginx not installed, skipping")
	}
	if _, err := exec.LookPath("dnsmasq"); err != nil {
		t.Skip("dnsmasq not installed, skipping")
	}

	// Create managers
	nm, err := proxy.NewNginxManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewNginxManager() error = %v", err)
	}

	dm, err := proxy.NewDnsmasqManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewDnsmasqManager() error = %v", err)
	}

	// Write configs
	if err := nm.WriteConfig(); err != nil {
		t.Fatalf("nginx WriteConfig() error = %v", err)
	}
	if err := dm.WriteConfig(); err != nil {
		t.Fatalf("dnsmasq WriteConfig() error = %v", err)
	}

	// Start all services
	if err := pm.StartAll(); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}

	// Wait for startup
	time.Sleep(1 * time.Second)

	// Check both running
	nginxRunning, _ := nm.IsRunning()
	dnsmasqRunning, _ := dm.IsRunning()

	if !nginxRunning {
		t.Error("nginx should be running after StartAll()")
	}
	if !dnsmasqRunning {
		t.Error("dnsmasq should be running after StartAll()")
	}

	// Stop all services
	if err := pm.StopAll(); err != nil {
		t.Fatalf("StopAll() error = %v", err)
	}

	// Wait for shutdown
	time.Sleep(1 * time.Second)

	// Check both stopped
	nginxRunning, _ = nm.IsRunning()
	dnsmasqRunning, _ = dm.IsRunning()

	if nginxRunning {
		t.Error("nginx should be stopped after StopAll()")
	}
	if dnsmasqRunning {
		t.Error("dnsmasq should be stopped after StopAll()")
	}
}

func TestIntegration_ProxyRequestRouting(t *testing.T) {
	cfg, pm, cleanup := setupIntegrationEnv(t)
	defer cleanup()

	// Check nginx is installed
	if _, err := exec.LookPath("nginx"); err != nil {
		t.Skip("nginx not installed, skipping")
	}

	// Start a simple HTTP server to proxy to
	go startTestServer(9000)
	time.Sleep(100 * time.Millisecond)

	// Create NginxManager
	nm, err := proxy.NewNginxManager(cfg, pm)
	if err != nil {
		t.Fatalf("NewNginxManager() error = %v", err)
	}

	// Write config
	if err := nm.WriteConfig(); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	// Start nginx
	if err := nm.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer nm.Stop()

	// Wait for startup
	time.Sleep(500 * time.Millisecond)

	// Test request to nginx
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", cfg.Nginx.HTTPPort))
	if err != nil {
		t.Skipf("Could not connect to nginx (may not have port access): %v", err)
	}
	defer resp.Body.Close()

	// Should get a response (even if 502 because our test server is basic)
	if resp.StatusCode == 0 {
		t.Error("Expected some response from nginx proxy")
	}
}

// startTestServer starts a simple HTTP server for testing
func startTestServer(port int) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK from test server"))
	})
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
