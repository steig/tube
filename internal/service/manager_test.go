package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewProcessManager(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pidDir := filepath.Join(tmpDir, "pids")

	pm, err := NewProcessManager(pidDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	if pm == nil {
		t.Fatal("NewProcessManager() returned nil")
	}

	// Verify directory was created
	if _, err := os.Stat(pidDir); os.IsNotExist(err) {
		t.Error("NewProcessManager() did not create pid directory")
	}
}

func TestNewProcessManager_ExistingDir(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Pre-create the pid directory
	pidDir := filepath.Join(tmpDir, "pids")
	if err := os.MkdirAll(pidDir, 0700); err != nil {
		t.Fatalf("failed to create pid dir: %v", err)
	}

	pm, err := NewProcessManager(pidDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	if pm == nil {
		t.Fatal("NewProcessManager() returned nil")
	}
}

func TestProcessManager_pidFilePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	tests := []struct {
		name string
		want string
	}{
		{"nginx", filepath.Join(tmpDir, "nginx.pid")},
		{"dnsmasq", filepath.Join(tmpDir, "dnsmasq.pid")},
		{"test-service", filepath.Join(tmpDir, "test-service.pid")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pm.pidFilePath(tt.name)
			if got != tt.want {
				t.Errorf("pidFilePath(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestProcessManager_writePID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	// Write a PID
	if err := pm.writePID("test", 12345); err != nil {
		t.Fatalf("writePID() error = %v", err)
	}

	// Verify file exists and has correct content
	pidFile := pm.pidFilePath("test")
	content, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("failed to read pid file: %v", err)
	}

	if string(content) != "12345\n" {
		t.Errorf("pid file content = %q, want %q", string(content), "12345\n")
	}
}

func TestProcessManager_readPID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	// Write a PID file manually
	pidFile := pm.pidFilePath("test")
	if err := os.WriteFile(pidFile, []byte("54321\n"), 0600); err != nil {
		t.Fatalf("failed to write test pid file: %v", err)
	}

	// Read it back
	pid, err := pm.readPID("test")
	if err != nil {
		t.Fatalf("readPID() error = %v", err)
	}

	if pid != 54321 {
		t.Errorf("readPID() = %d, want %d", pid, 54321)
	}
}

func TestProcessManager_readPID_NotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	_, err = pm.readPID("nonexistent")
	if err == nil {
		t.Error("readPID() expected error for non-existent pid file, got nil")
	}
}

func TestProcessManager_readPID_InvalidContent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	// Write invalid content
	pidFile := pm.pidFilePath("test")
	if err := os.WriteFile(pidFile, []byte("not-a-number\n"), 0600); err != nil {
		t.Fatalf("failed to write test pid file: %v", err)
	}

	_, err = pm.readPID("test")
	if err == nil {
		t.Error("readPID() expected error for invalid pid content, got nil")
	}
}

func TestProcessManager_cleanupPID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	// Write a PID file
	if err := pm.writePID("test", 12345); err != nil {
		t.Fatalf("writePID() error = %v", err)
	}

	// Cleanup
	if err := pm.cleanupPID("test"); err != nil {
		t.Fatalf("cleanupPID() error = %v", err)
	}

	// Verify file is gone
	pidFile := pm.pidFilePath("test")
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("cleanupPID() did not remove pid file")
	}
}

func TestProcessManager_cleanupPID_NotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	// Should not error on non-existent file
	if err := pm.cleanupPID("nonexistent"); err != nil {
		t.Errorf("cleanupPID() error = %v, want nil", err)
	}
}

func TestProcessManager_getServiceConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	tests := []struct {
		name       string
		wantBinary string
		wantArgs   []string
		wantErr    bool
	}{
		{
			name:       "nginx",
			wantBinary: "nginx",
			wantArgs:   []string{"-g", "daemon off;"},
			wantErr:    false,
		},
		{
			name:       "dnsmasq",
			wantBinary: "dnsmasq",
			wantArgs:   []string{},
			wantErr:    false,
		},
		{
			name:    "unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binary, args, err := pm.getServiceConfig(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("getServiceConfig(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if binary != tt.wantBinary {
					t.Errorf("getServiceConfig(%q) binary = %q, want %q", tt.name, binary, tt.wantBinary)
				}
				if len(args) != len(tt.wantArgs) {
					t.Errorf("getServiceConfig(%q) args = %v, want %v", tt.name, args, tt.wantArgs)
				}
			}
		})
	}
}

func TestProcessManager_GetServicePID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	// Test with no PID file
	if pid := pm.GetServicePID("nonexistent"); pid != 0 {
		t.Errorf("GetServicePID() = %d, want 0 for non-existent", pid)
	}

	// Write a PID file
	if err := pm.writePID("test", 99999); err != nil {
		t.Fatalf("writePID() error = %v", err)
	}

	if pid := pm.GetServicePID("test"); pid != 99999 {
		t.Errorf("GetServicePID() = %d, want 99999", pid)
	}
}

func TestProcessManager_Status_NotRunning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	status, err := pm.Status("test")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if status != "stopped" {
		t.Errorf("Status() = %q, want %q", status, "stopped")
	}
}

func TestProcessManager_IsRunning_NotRunning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "tube-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm, err := NewProcessManager(tmpDir)
	if err != nil {
		t.Fatalf("NewProcessManager() error = %v", err)
	}

	// Test with no PID file
	isRunning, err := pm.IsRunning("test")
	if err != nil {
		t.Fatalf("IsRunning() error = %v", err)
	}
	if isRunning {
		t.Error("IsRunning() = true, want false for non-existent PID file")
	}

	// Test with stale PID file (process doesn't exist)
	// Use PID 1000000000 which almost certainly doesn't exist
	if err := pm.writePID("stale", 1000000000); err != nil {
		t.Fatalf("writePID() error = %v", err)
	}

	isRunning, err = pm.IsRunning("stale")
	if err != nil {
		t.Fatalf("IsRunning() error = %v", err)
	}
	if isRunning {
		t.Error("IsRunning() = true, want false for stale PID")
	}

	// Verify stale PID file was cleaned up
	pidFile := pm.pidFilePath("stale")
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("IsRunning() did not clean up stale PID file")
	}
}
