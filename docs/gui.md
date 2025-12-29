# GUI Guide

tube provides two graphical interfaces: a **Menu Bar App** and a **Web Dashboard**.

## Starting the GUI

Run the GUI app:

```bash
tube-gui
```

This starts both:
1. A system tray icon in your macOS menu bar
2. A web dashboard at `http://localhost:3249`

The app runs in the foreground. To run in the background, use:

```bash
tube-gui &
```

Or add it to your login items for automatic startup.

---

## Menu Bar App

### Overview

The menu bar app provides quick access to tube functionality without leaving your current workflow.

```
┌───────────────────────────────┐
│ tube ●                        │  ← Status indicator in menu bar
├───────────────────────────────┤
│ ● Services Running            │  ← Current status
├───────────────────────────────┤
│ Projects                     ▸│  ← Expandable submenu
│   ├─ frontend.test    :3000   │
│   ├─ api.test         :8080   │
│   └─ dashboard.test   :4200   │
├───────────────────────────────┤
│ Start Services                │  ← Start nginx + dnsmasq
│ Stop Services                 │  ← Stop all services
├───────────────────────────────┤
│ Open Dashboard...             │  ← Opens web UI in browser
│ Quit tube                     │  ← Exit the app
└───────────────────────────────┘
```

### Status Indicators

The menu bar icon shows the current service state:

| Icon | State | Meaning |
|------|-------|---------|
| `tube ●` | Running | Both nginx and dnsmasq are running |
| `tube ◐` | Partial | Only one service is running |
| `tube ○` | Stopped | All services are stopped |

### Projects Submenu

Click on any project to open it in your default browser:

- `frontend.test :3000` → Opens `http://frontend.test`
- `api.test :8080` → Opens `http://api.test`

### Menu Actions

| Action | Description |
|--------|-------------|
| **Start Services** | Writes configs and starts nginx + dnsmasq |
| **Stop Services** | Stops all running services |
| **Open Dashboard...** | Opens the web dashboard in your browser |
| **Quit tube** | Exits the menu bar app |

### Notifications

The app shows system notifications for:
- Services started successfully
- Services stopped
- Errors (failed to start, configuration issues)

---

## Web Dashboard

### Accessing the Dashboard

Open in your browser:
```
http://localhost:3249
```

Or click "Open Dashboard..." in the menu bar app.

### Dashboard Layout

```
╭──────────────────────────────────────────────────────────────────╮
│                                                                  │
│   tube Dashboard                              ● Running          │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Services                                                       │
│   ┌─────────────────────┐  ┌─────────────────────┐              │
│   │ ● nginx             │  │ ● dnsmasq           │              │
│   │   running (pid 123) │  │   running (pid 456) │              │
│   └─────────────────────┘  └─────────────────────┘              │
│                                                                  │
│   [Start Services]  [Stop Services]                              │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Projects                                                       │
│   ┌────────────────────────────────────────────────────────────┐│
│   │ ● frontend    http://frontend.test          :3000    [x]   ││
│   │ ○ api         http://api.test               :8080    [x]   ││
│   │ ● dashboard   http://dashboard.test         :4200    [x]   ││
│   └────────────────────────────────────────────────────────────┘│
│                                                                  │
│   [________name________] [__port__] [Add Project]                │
│                                                                  │
╰──────────────────────────────────────────────────────────────────╯
```

### Services Section

Shows the status of nginx and dnsmasq:
- **Running** (green dot) - Service is active with PID
- **Stopped** (grey dot) - Service is not running

**Controls:**
- **Start Services** - Starts both nginx and dnsmasq
- **Stop Services** - Stops all services

### Projects Section

Lists all configured projects with:
- **Status indicator** - Green if port is responding, grey if not
- **Name** - Project name
- **URL** - Click to open in browser
- **Port** - Local port number
- **Delete button** - Remove the project

**Add New Project:**
1. Enter project name (e.g., `myapp`)
2. Enter port number (e.g., `3000`)
3. Click "Add Project"
4. Services are automatically restarted

---

## Dashboard API

The dashboard exposes a REST API that you can use programmatically.

### Base URL

```
http://localhost:3249/api
```

### Endpoints

#### GET /api/status

Get the current status of all services.

**Response:**
```json
{
  "nginx": {
    "name": "nginx",
    "running": true,
    "pid": 12345
  },
  "dnsmasq": {
    "name": "dnsmasq",
    "running": true,
    "pid": 12346
  }
}
```

#### GET /api/projects

Get all configured projects.

**Response:**
```json
{
  "projects": [
    {
      "name": "frontend",
      "port": 3000,
      "url": "http://frontend.test",
      "running": true
    },
    {
      "name": "api",
      "port": 8080,
      "url": "http://api.test",
      "running": false
    }
  ]
}
```

#### POST /api/services/start

Start all services.

**Response:**
```json
{
  "success": true,
  "message": "Services started"
}
```

#### POST /api/services/stop

Stop all services.

**Response:**
```json
{
  "success": true,
  "message": "Services stopped"
}
```

#### POST /api/project/add

Add a new project.

**Request:**
```json
{
  "name": "myapp",
  "port": 3000
}
```

**Response:**
```json
{
  "success": true,
  "message": "Project added"
}
```

#### POST /api/project/remove

Remove a project.

**Request:**
```json
{
  "name": "myapp"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Project removed"
}
```

### API Examples

**Using curl:**

```bash
# Get status
curl http://localhost:3249/api/status

# Get projects
curl http://localhost:3249/api/projects

# Start services
curl -X POST http://localhost:3249/api/services/start

# Stop services
curl -X POST http://localhost:3249/api/services/stop

# Add a project
curl -X POST http://localhost:3249/api/project/add \
  -H "Content-Type: application/json" \
  -d '{"name": "myapp", "port": 3000}'

# Remove a project
curl -X POST http://localhost:3249/api/project/remove \
  -H "Content-Type: application/json" \
  -d '{"name": "myapp"}'
```

---

## Auto-Start on Login

### Using launchd (Recommended)

Create a launch agent:

```bash
cat > ~/Library/LaunchAgents/com.tube.gui.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.tube.gui</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/tube-gui</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>
EOF

# Load the agent
launchctl load ~/Library/LaunchAgents/com.tube.gui.plist
```

To disable auto-start:

```bash
launchctl unload ~/Library/LaunchAgents/com.tube.gui.plist
rm ~/Library/LaunchAgents/com.tube.gui.plist
```

### Using Login Items

1. Open System Preferences → Users & Groups
2. Select your user
3. Click "Login Items"
4. Click "+" and add `/usr/local/bin/tube-gui`

---

## Troubleshooting GUI

### Menu bar icon doesn't appear

The systray library requires a running display server. Ensure:
1. You're running on macOS (not in SSH or headless mode)
2. The app has permission to display in the menu bar

### Dashboard not loading

1. Check if the port is in use:
   ```bash
   lsof -i :3249
   ```

2. Try a different port in config:
   ```yaml
   proxy:
     dashboard_port: 3250
   ```

### Projects not updating

Click "Refresh" or reload the page. The dashboard doesn't auto-refresh (yet).

### Notifications not appearing

Check System Preferences → Notifications → tube-gui and ensure notifications are enabled.

---

## Keyboard Shortcuts

The menu bar app supports these keyboard shortcuts when the menu is open:

| Shortcut | Action |
|----------|--------|
| `↑` / `↓` | Navigate menu items |
| `Enter` | Select item |
| `Esc` | Close menu |

---

## Future Features

Planned improvements for the GUI:
- [ ] Auto-refresh dashboard
- [ ] Project status websocket updates
- [ ] Log viewer
- [ ] Configuration editor
- [ ] Service restart buttons
- [ ] Project sorting and filtering
