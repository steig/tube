package gui

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/steig/tube/internal/cli"
	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/proxy"
	"github.com/steig/tube/internal/service"
)

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
	// Create ProcessManager
	pm, err := service.NewProcessManager(cfg.Directories.PIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create process manager: %w", err)
	}

	// Create NginxManager
	ngx, err := proxy.NewNginxManager(cfg, pm)
	if err != nil {
		return nil, fmt.Errorf("failed to create nginx manager: %w", err)
	}

	// Create DnsmasqManager
	dms, err := proxy.NewDnsmasqManager(cfg, pm)
	if err != nil {
		return nil, fmt.Errorf("failed to create dnsmasq manager: %w", err)
	}

	return &Dashboard{
		cfg:        cfg,
		configPath: configPath,
		pm:         pm,
		ngx:        ngx,
		dms:        dms,
	}, nil
}

// Start starts the dashboard server
func (d *Dashboard) Start() error {
	mux := http.NewServeMux()

	// Routes
	mux.HandleFunc("/", d.handleIndex)
	mux.HandleFunc("/api/status", d.handleAPIStatus)
	mux.HandleFunc("/api/projects", d.handleAPIProjects)
	mux.HandleFunc("/api/services/start", d.handleAPIStart)
	mux.HandleFunc("/api/services/stop", d.handleAPIStop)
	mux.HandleFunc("/api/project/add", d.handleAPIAddProject)
	mux.HandleFunc("/api/project/remove", d.handleAPIRemoveProject)

	addr := fmt.Sprintf("127.0.0.1:%d", d.cfg.Proxy.DashboardPort)
	d.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return d.server.ListenAndServe()
}

// Stop stops the dashboard server
func (d *Dashboard) Stop() error {
	if d.server != nil {
		return d.server.Close()
	}
	return nil
}

// handleIndex serves the main dashboard page
func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
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

// handleAPIStatus returns the current service status
func (d *Dashboard) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	nginxRunning, _ := d.ngx.IsRunning()
	nginxStatus, _ := d.ngx.Status()
	dnsmasqRunning, _ := d.dms.IsRunning()
	dnsmasqStatus, _ := d.dms.Status()

	resp := StatusResponse{
		Nginx: ServiceStatus{
			Running: nginxRunning,
			Status:  nginxStatus,
		},
		Dnsmasq: ServiceStatus{
			Running: dnsmasqRunning,
			Status:  dnsmasqStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleAPIProjects returns the list of projects
func (d *Dashboard) handleAPIProjects(w http.ResponseWriter, r *http.Request) {
	projects, _ := cli.ListProjects(d.cfg)

	var resp []ProjectResponse
	for _, p := range projects {
		resp = append(resp, ProjectResponse{
			Name:    p.Name,
			Port:    p.Port,
			URL:     p.LocalURL,
			Running: p.Running,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleAPIStart starts the services
func (d *Dashboard) handleAPIStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Write configs
	if err := d.ngx.WriteConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := d.dms.WriteConfig(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Start services
	if err := d.pm.StartAll(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAPIStop stops the services
func (d *Dashboard) handleAPIStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := d.pm.StopAll(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAPIAddProject adds a new project
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAPIRemoveProject removes a project
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// dashboardHTML is the embedded dashboard HTML
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>tube - Dashboard</title>
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #fff;
            min-height: 100vh;
            padding: 2rem;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
        }
        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 2rem;
        }
        h1 {
            font-size: 2rem;
            font-weight: 600;
        }
        h1 span {
            color: #4ade80;
        }
        .status-badge {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.5rem 1rem;
            border-radius: 9999px;
            font-size: 0.875rem;
            font-weight: 500;
        }
        .status-badge.running {
            background: rgba(74, 222, 128, 0.2);
            color: #4ade80;
        }
        .status-badge.stopped {
            background: rgba(248, 113, 113, 0.2);
            color: #f87171;
        }
        .status-badge .dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: currentColor;
        }
        .card {
            background: rgba(255, 255, 255, 0.05);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 12px;
            padding: 1.5rem;
            margin-bottom: 1.5rem;
        }
        .card h2 {
            font-size: 1.125rem;
            font-weight: 600;
            margin-bottom: 1rem;
            color: #94a3b8;
        }
        .services {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 1rem;
        }
        .service {
            display: flex;
            align-items: center;
            gap: 0.75rem;
        }
        .service .indicator {
            width: 12px;
            height: 12px;
            border-radius: 50%;
        }
        .service .indicator.running {
            background: #4ade80;
            box-shadow: 0 0 8px #4ade80;
        }
        .service .indicator.stopped {
            background: #64748b;
        }
        .service .name {
            font-weight: 500;
        }
        .service .status {
            color: #64748b;
            font-size: 0.875rem;
        }
        .controls {
            display: flex;
            gap: 1rem;
            margin-top: 1rem;
        }
        button {
            padding: 0.75rem 1.5rem;
            border: none;
            border-radius: 8px;
            font-size: 0.875rem;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
        }
        button.primary {
            background: #4ade80;
            color: #000;
        }
        button.primary:hover {
            background: #22c55e;
        }
        button.secondary {
            background: rgba(255, 255, 255, 0.1);
            color: #fff;
        }
        button.secondary:hover {
            background: rgba(255, 255, 255, 0.2);
        }
        button:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        .projects-list {
            display: flex;
            flex-direction: column;
            gap: 0.75rem;
        }
        .project {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 1rem;
            background: rgba(255, 255, 255, 0.03);
            border-radius: 8px;
        }
        .project-info {
            display: flex;
            align-items: center;
            gap: 1rem;
        }
        .project-info .indicator {
            width: 8px;
            height: 8px;
            border-radius: 50%;
        }
        .project-info .indicator.running {
            background: #4ade80;
        }
        .project-info .indicator.stopped {
            background: #64748b;
        }
        .project-name {
            font-weight: 500;
        }
        .project-url {
            color: #60a5fa;
            text-decoration: none;
            font-size: 0.875rem;
        }
        .project-url:hover {
            text-decoration: underline;
        }
        .project-port {
            color: #64748b;
            font-size: 0.875rem;
        }
        .project-actions button {
            padding: 0.5rem 1rem;
            font-size: 0.75rem;
        }
        .add-project {
            display: flex;
            gap: 0.5rem;
            margin-top: 1rem;
        }
        input {
            padding: 0.75rem 1rem;
            border: 1px solid rgba(255, 255, 255, 0.2);
            border-radius: 8px;
            background: rgba(255, 255, 255, 0.05);
            color: #fff;
            font-size: 0.875rem;
        }
        input::placeholder {
            color: #64748b;
        }
        input:focus {
            outline: none;
            border-color: #4ade80;
        }
        .empty-state {
            text-align: center;
            padding: 2rem;
            color: #64748b;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1><span>tube</span> Dashboard</h1>
            <div id="overall-status" class="status-badge stopped">
                <span class="dot"></span>
                <span>Loading...</span>
            </div>
        </header>

        <div class="card">
            <h2>Services</h2>
            <div class="services" id="services">
                <div class="service">
                    <div class="indicator" id="nginx-indicator"></div>
                    <div>
                        <div class="name">nginx</div>
                        <div class="status" id="nginx-status">checking...</div>
                    </div>
                </div>
                <div class="service">
                    <div class="indicator" id="dnsmasq-indicator"></div>
                    <div>
                        <div class="name">dnsmasq</div>
                        <div class="status" id="dnsmasq-status">checking...</div>
                    </div>
                </div>
            </div>
            <div class="controls">
                <button class="primary" id="start-btn" onclick="startServices()">Start Services</button>
                <button class="secondary" id="stop-btn" onclick="stopServices()">Stop Services</button>
            </div>
        </div>

        <div class="card">
            <h2>Projects</h2>
            <div class="projects-list" id="projects-list">
                <div class="empty-state">Loading projects...</div>
            </div>
            <div class="add-project">
                <input type="text" id="new-name" placeholder="Project name" style="flex: 1;">
                <input type="number" id="new-port" placeholder="Port" style="width: 100px;">
                <button class="primary" onclick="addProject()">Add Project</button>
            </div>
        </div>
    </div>

    <script>
        async function fetchStatus() {
            try {
                const res = await fetch('/api/status');
                const data = await res.json();

                // Update nginx
                const nginxIndicator = document.getElementById('nginx-indicator');
                const nginxStatus = document.getElementById('nginx-status');
                nginxIndicator.className = 'indicator ' + (data.nginx.running ? 'running' : 'stopped');
                nginxStatus.textContent = data.nginx.status;

                // Update dnsmasq
                const dnsmasqIndicator = document.getElementById('dnsmasq-indicator');
                const dnsmasqStatus = document.getElementById('dnsmasq-status');
                dnsmasqIndicator.className = 'indicator ' + (data.dnsmasq.running ? 'running' : 'stopped');
                dnsmasqStatus.textContent = data.dnsmasq.status;

                // Update overall status
                const overall = document.getElementById('overall-status');
                if (data.nginx.running && data.dnsmasq.running) {
                    overall.className = 'status-badge running';
                    overall.innerHTML = '<span class="dot"></span><span>Running</span>';
                } else {
                    overall.className = 'status-badge stopped';
                    overall.innerHTML = '<span class="dot"></span><span>Stopped</span>';
                }

                // Update buttons
                document.getElementById('start-btn').disabled = data.nginx.running && data.dnsmasq.running;
                document.getElementById('stop-btn').disabled = !data.nginx.running && !data.dnsmasq.running;
            } catch (e) {
                console.error('Failed to fetch status:', e);
            }
        }

        async function fetchProjects() {
            try {
                const res = await fetch('/api/projects');
                const projects = await res.json();

                const list = document.getElementById('projects-list');
                if (!projects || projects.length === 0) {
                    list.innerHTML = '<div class="empty-state">No projects configured. Add one below!</div>';
                    return;
                }

                list.innerHTML = projects.map(p => ` + "`" + `
                    <div class="project">
                        <div class="project-info">
                            <div class="indicator ${p.running ? 'running' : 'stopped'}"></div>
                            <div>
                                <div class="project-name">${p.name}</div>
                                <a class="project-url" href="${p.url}" target="_blank">${p.url}</a>
                            </div>
                        </div>
                        <div style="display: flex; align-items: center; gap: 1rem;">
                            <span class="project-port">:${p.port}</span>
                            <button class="secondary project-actions" onclick="removeProject('${p.name}')">Remove</button>
                        </div>
                    </div>
                ` + "`" + `).join('');
            } catch (e) {
                console.error('Failed to fetch projects:', e);
            }
        }

        async function startServices() {
            try {
                await fetch('/api/services/start', { method: 'POST' });
                fetchStatus();
            } catch (e) {
                alert('Failed to start services: ' + e.message);
            }
        }

        async function stopServices() {
            try {
                await fetch('/api/services/stop', { method: 'POST' });
                fetchStatus();
            } catch (e) {
                alert('Failed to stop services: ' + e.message);
            }
        }

        async function addProject() {
            const name = document.getElementById('new-name').value.trim();
            const port = parseInt(document.getElementById('new-port').value);

            if (!name || !port) {
                alert('Please enter both name and port');
                return;
            }

            try {
                const res = await fetch('/api/project/add', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name, port })
                });

                if (!res.ok) {
                    const err = await res.text();
                    alert('Failed to add project: ' + err);
                    return;
                }

                document.getElementById('new-name').value = '';
                document.getElementById('new-port').value = '';
                fetchProjects();
            } catch (e) {
                alert('Failed to add project: ' + e.message);
            }
        }

        async function removeProject(name) {
            if (!confirm('Remove project "' + name + '"?')) return;

            try {
                const res = await fetch('/api/project/remove', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name })
                });

                if (!res.ok) {
                    const err = await res.text();
                    alert('Failed to remove project: ' + err);
                    return;
                }

                fetchProjects();
            } catch (e) {
                alert('Failed to remove project: ' + e.message);
            }
        }

        // Initial fetch
        fetchStatus();
        fetchProjects();

        // Poll for updates
        setInterval(fetchStatus, 5000);
    </script>
</body>
</html>
`
