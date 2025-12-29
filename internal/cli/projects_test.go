package cli

import (
	"testing"

	"github.com/steig/tube/internal/config"
)

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple name",
			input:   "myapp",
			wantErr: false,
		},
		{
			name:    "valid with hyphens",
			input:   "my-app-name",
			wantErr: false,
		},
		{
			name:    "valid with numbers",
			input:   "app123",
			wantErr: false,
		},
		{
			name:    "valid single character",
			input:   "a",
			wantErr: false,
		},
		{
			name:    "valid two characters",
			input:   "ab",
			wantErr: false,
		},
		{
			name:    "empty name",
			input:   "",
			wantErr: true,
			errMsg:  "project name cannot be empty",
		},
		{
			name:    "too long (64 chars)",
			input:   "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijkl",
			wantErr: true,
		},
		{
			name:    "starts with hyphen",
			input:   "-myapp",
			wantErr: true,
		},
		{
			name:    "ends with hyphen",
			input:   "myapp-",
			wantErr: true,
		},
		{
			name:    "contains underscore",
			input:   "my_app",
			wantErr: true,
		},
		{
			name:    "contains space",
			input:   "my app",
			wantErr: true,
		},
		{
			name:    "contains dot",
			input:   "my.app",
			wantErr: true,
		},
		{
			name:    "uppercase allowed",
			input:   "MyApp",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProjectName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{
			name:    "valid port 3000",
			port:    3000,
			wantErr: false,
		},
		{
			name:    "valid port 8080",
			port:    8080,
			wantErr: false,
		},
		{
			name:    "valid port min boundary (1024)",
			port:    1024,
			wantErr: false,
		},
		{
			name:    "valid port max boundary (65535)",
			port:    65535,
			wantErr: false,
		},
		{
			name:    "invalid port too low (1023)",
			port:    1023,
			wantErr: true,
		},
		{
			name:    "invalid port too low (80)",
			port:    80,
			wantErr: true,
		},
		{
			name:    "invalid port too high (65536)",
			port:    65536,
			wantErr: true,
		},
		{
			name:    "invalid port zero",
			port:    0,
			wantErr: true,
		},
		{
			name:    "invalid port negative",
			port:    -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%d) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:    "valid port string",
			input:   "3000",
			want:    3000,
			wantErr: false,
		},
		{
			name:    "valid port string with spaces",
			input:   "8080",
			want:    8080,
			wantErr: false,
		},
		{
			name:    "invalid non-numeric",
			input:   "abc",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid empty",
			input:   "",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid port range",
			input:   "80",
			want:    0,
			wantErr: true,
		},
		{
			name:    "invalid float",
			input:   "3000.5",
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePort(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePort(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestProjectExists(t *testing.T) {
	cfg := &config.Config{
		Projects: map[string]int{
			"myapp": 3000,
			"api":   8080,
		},
	}

	tests := []struct {
		name string
		want bool
	}{
		{"myapp", true},
		{"api", true},
		{"notexist", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProjectExists(cfg, tt.name)
			if got != tt.want {
				t.Errorf("ProjectExists(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestPortExists(t *testing.T) {
	cfg := &config.Config{
		Projects: map[string]int{
			"myapp": 3000,
			"api":   8080,
		},
	}

	tests := []struct {
		port int
		want bool
	}{
		{3000, true},
		{8080, true},
		{4000, false},
		{0, false},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := PortExists(cfg, tt.port)
			if got != tt.want {
				t.Errorf("PortExists(%d) = %v, want %v", tt.port, got, tt.want)
			}
		})
	}
}

func TestGetProjectPort(t *testing.T) {
	cfg := &config.Config{
		Projects: map[string]int{
			"myapp": 3000,
			"api":   8080,
		},
	}

	tests := []struct {
		name    string
		want    int
		wantErr bool
	}{
		{"myapp", 3000, false},
		{"api", 8080, false},
		{"notexist", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetProjectPort(cfg, tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProjectPort(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetProjectPort(%q) = %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestListProjects(t *testing.T) {
	cfg := &config.Config{
		Proxy: config.ProxyConfig{
			LocalDomain: ".test",
		},
		Projects: map[string]int{
			"myapp": 3000,
			"api":   8080,
		},
	}

	statuses, err := ListProjects(cfg)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}

	if len(statuses) != 2 {
		t.Errorf("ListProjects() returned %d statuses, want 2", len(statuses))
	}

	// Check that all projects are in the list
	found := make(map[string]bool)
	for _, s := range statuses {
		found[s.Name] = true

		// Verify LocalURL format
		expectedURL := "http://" + s.Name + ".test"
		if s.LocalURL != expectedURL {
			t.Errorf("LocalURL = %q, want %q", s.LocalURL, expectedURL)
		}
	}

	if !found["myapp"] {
		t.Error("ListProjects() missing 'myapp'")
	}
	if !found["api"] {
		t.Error("ListProjects() missing 'api'")
	}
}

func TestListProjects_Empty(t *testing.T) {
	cfg := &config.Config{
		Projects: map[string]int{},
	}

	statuses, err := ListProjects(cfg)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}

	if len(statuses) != 0 {
		t.Errorf("ListProjects() returned %d statuses for empty config, want 0", len(statuses))
	}
}
