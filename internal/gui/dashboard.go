//go:build darwin

package gui

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"time"

	"github.com/steig/tube/internal/cli"
	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/proxy"
	"github.com/steig/tube/internal/service"
)

//go:embed templates/dashboard.html
var dashboardFS embed.FS

var dashboardTmpl = template.Must(template.ParseFS(dashboardFS, "templates/dashboard.html"))

// Dashboard represents the web dashboard server
type Dashboard struct {
	cfg        *config.Config
	configPath string
	pm         *service.ProcessManager
	ngx        *proxy.NginxManager
	dms        *proxy.DnsmasqManager
	server     *http.Server
}

// NewDashboard creates a new dashboard server
func NewDashboard(cfg *config.Config, configPath string) (*Dashboard, error) {
	pm, err := service.NewProcessManager(cfg.Directories.PIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create process manager: %w", err)
	}

	ngx, err := proxy.NewNginxManager(cfg, pm)
	if err != nil {
		return nil, fmt.Errorf("failed to create nginx manager: %w", err)
	}

	dms, err := proxy.NewDnsmasqManager(cfg, pm)
	if err != nil {
		return nil, fmt.Errorf("failed to create dnsmasq manager: %w", err)
	}

	return &Dashboard{cfg: cfg, configPath: configPath, pm: pm, ngx: ngx, dms: dms}, nil
}

// Start binds the dashboard port, then serves until Stop is called.
// Bind failures are surfaced immediately so callers know the dashboard isn't
// actually listening.
func (d *Dashboard) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", d.handleIndex)
	mux.HandleFunc("/api/status", d.handleAPIStatus)
	mux.HandleFunc("/api/projects", d.handleAPIProjects)
	mux.HandleFunc("/api/services/start", d.handleAPIStart)
	mux.HandleFunc("/api/services/stop", d.handleAPIStop)
	mux.HandleFunc("/api/project/add", d.handleAPIAddProject)
	mux.HandleFunc("/api/project/remove", d.handleAPIRemoveProject)

	addr := fmt.Sprintf("127.0.0.1:%d", d.cfg.Proxy.DashboardPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("dashboard: failed to bind %s: %w", addr, err)
	}

	d.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return d.server.Serve(listener)
}

// Stop stops the dashboard server.
func (d *Dashboard) Stop() error {
	if d.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return d.server.Shutdown(ctx)
}

// handleIndex serves the main dashboard page. Rendered via html/template so any
// future template variables get auto-escaped.
func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dashboardTmpl.Execute(w, nil); err != nil {
		http.Error(w, "failed to render dashboard", http.StatusInternalServerError)
	}
}

// StatusResponse represents the API status response
type StatusResponse struct {
	Nginx   ServiceStatus `json:"nginx"`
	Dnsmasq ServiceStatus `json:"dnsmasq"`
}

// ServiceStatus represents a service status
type ServiceStatus struct {
	Running bool   `json:"running"`
	Status  string `json:"status"`
}

// ProjectResponse represents a project in API responses
type ProjectResponse struct {
	Name    string `json:"name"`
	Port    int    `json:"port"`
	URL     string `json:"url"`
	Running bool   `json:"running"`
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (d *Dashboard) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	nginxRunning, _ := d.ngx.IsRunning()
	nginxStatus, _ := d.ngx.Status()
	dnsmasqRunning, _ := d.dms.IsRunning()
	dnsmasqStatus, _ := d.dms.Status()

	writeJSON(w, StatusResponse{
		Nginx:   ServiceStatus{Running: nginxRunning, Status: nginxStatus},
		Dnsmasq: ServiceStatus{Running: dnsmasqRunning, Status: dnsmasqStatus},
	})
}

func (d *Dashboard) handleAPIProjects(w http.ResponseWriter, r *http.Request) {
	projects, _ := cli.ListProjects(d.cfg)
	resp := make([]ProjectResponse, 0, len(projects))
	for _, p := range projects {
		resp = append(resp, ProjectResponse{
			Name: p.Name, Port: p.Port, URL: p.LocalURL, Running: p.Running,
		})
	}
	writeJSON(w, resp)
}

func (d *Dashboard) handleAPIStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := d.ngx.WriteConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := d.dms.WriteConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := d.pm.StartAll(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (d *Dashboard) handleAPIStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := d.pm.StopAll(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (d *Dashboard) handleAPIAddProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
		Port int    `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := cli.AddProject(d.cfg, d.configPath, d.pm, d.ngx, d.dms, req.Name, req.Port); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (d *Dashboard) handleAPIRemoveProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := cli.RemoveProject(d.cfg, d.configPath, d.pm, d.ngx, d.dms, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}
