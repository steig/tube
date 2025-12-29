// Package gui provides the graphical user interface components for tube
package gui

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/proxy"
	"github.com/steig/tube/internal/service"
)

// TrayApp represents the system tray application
type TrayApp struct {
	cfg        *config.Config
	configPath string
	pm         *service.ProcessManager
	ngx        *proxy.NginxManager
	dms        *proxy.DnsmasqManager

	// Menu items
	mStatus      *systray.MenuItem
	mProjects    *systray.MenuItem
	mStart       *systray.MenuItem
	mStop        *systray.MenuItem
	mDashboard   *systray.MenuItem
	mQuit        *systray.MenuItem
	projectItems []*systray.MenuItem
}

// NewTrayApp creates a new tray application
func NewTrayApp(cfg *config.Config, configPath string) (*TrayApp, error) {
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

	return &TrayApp{
		cfg:        cfg,
		configPath: configPath,
		pm:         pm,
		ngx:        ngx,
		dms:        dms,
	}, nil
}

// Run starts the system tray application
func (t *TrayApp) Run() {
	systray.Run(t.onReady, t.onExit)
}

// onReady is called when the systray is ready
func (t *TrayApp) onReady() {
	// Set icon and title
	systray.SetIcon(getTrayIcon())
	systray.SetTitle("tube")
	systray.SetTooltip("tube - Local Development Proxy")

	// Status section
	t.mStatus = systray.AddMenuItem("● tube", "Status")
	t.mStatus.Disable()
	systray.AddSeparator()

	// Projects section
	t.mProjects = systray.AddMenuItem("Projects", "Configured projects")
	t.mProjects.Disable()
	t.updateProjectMenu()
	systray.AddSeparator()

	// Control section
	t.mStart = systray.AddMenuItem("Start Services", "Start nginx and dnsmasq")
	t.mStop = systray.AddMenuItem("Stop Services", "Stop all services")
	systray.AddSeparator()

	// Dashboard
	t.mDashboard = systray.AddMenuItem("Open Dashboard...", "Open web dashboard")
	systray.AddSeparator()

	// Quit
	t.mQuit = systray.AddMenuItem("Quit tube", "Quit the application")

	// Update initial status
	t.updateStatus()

	// Handle clicks
	go t.handleClicks()
}

// onExit is called when the systray is exited
func (t *TrayApp) onExit() {
	// Cleanup if needed
}

// handleClicks handles menu item clicks
func (t *TrayApp) handleClicks() {
	for {
		select {
		case <-t.mStart.ClickedCh:
			t.startServices()
		case <-t.mStop.ClickedCh:
			t.stopServices()
		case <-t.mDashboard.ClickedCh:
			t.openDashboard()
		case <-t.mQuit.ClickedCh:
			systray.Quit()
		}
	}
}

// updateStatus updates the status display
func (t *TrayApp) updateStatus() {
	nginxRunning, _ := t.ngx.IsRunning()
	dnsmasqRunning, _ := t.dms.IsRunning()

	if nginxRunning && dnsmasqRunning {
		systray.SetTitle("tube ●")
		t.mStatus.SetTitle("● Services Running")
		t.mStart.Disable()
		t.mStop.Enable()
	} else if nginxRunning || dnsmasqRunning {
		systray.SetTitle("tube ◐")
		t.mStatus.SetTitle("◐ Partially Running")
		t.mStart.Enable()
		t.mStop.Enable()
	} else {
		systray.SetTitle("tube ○")
		t.mStatus.SetTitle("○ Services Stopped")
		t.mStart.Enable()
		t.mStop.Disable()
	}
}

// updateProjectMenu updates the project submenu
func (t *TrayApp) updateProjectMenu() {
	// Clear existing project items
	for _, item := range t.projectItems {
		item.Hide()
	}
	t.projectItems = nil

	// Add projects
	if len(t.cfg.Projects) == 0 {
		item := t.mProjects.AddSubMenuItem("No projects configured", "")
		item.Disable()
		t.projectItems = append(t.projectItems, item)
	} else {
		for name, port := range t.cfg.Projects {
			url := fmt.Sprintf("http://%s.test", name)
			title := fmt.Sprintf("%s :%d", name, port)
			item := t.mProjects.AddSubMenuItem(title, url)
			t.projectItems = append(t.projectItems, item)

			// Handle click to open in browser
			go func(menuItem *systray.MenuItem, projectURL string) {
				for range menuItem.ClickedCh {
					openBrowser(projectURL)
				}
			}(item, url)
		}
	}
}

// startServices starts nginx and dnsmasq
func (t *TrayApp) startServices() {
	// Write configs first
	if err := t.ngx.WriteConfig(); err != nil {
		t.showNotification("Error", fmt.Sprintf("Failed to write nginx config: %v", err))
		return
	}

	if err := t.dms.WriteConfig(); err != nil {
		t.showNotification("Error", fmt.Sprintf("Failed to write dnsmasq config: %v", err))
		return
	}

	// Start services
	if err := t.pm.StartAll(); err != nil {
		t.showNotification("Error", fmt.Sprintf("Failed to start services: %v", err))
		return
	}

	t.updateStatus()
	t.showNotification("tube", "Services started")
}

// stopServices stops all services
func (t *TrayApp) stopServices() {
	if err := t.pm.StopAll(); err != nil {
		t.showNotification("Error", fmt.Sprintf("Failed to stop services: %v", err))
		return
	}

	t.updateStatus()
	t.showNotification("tube", "Services stopped")
}

// openDashboard opens the web dashboard
func (t *TrayApp) openDashboard() {
	url := fmt.Sprintf("http://localhost:%d", t.cfg.Proxy.DashboardPort)
	openBrowser(url)
}

// showNotification shows a system notification
func (t *TrayApp) showNotification(title, message string) {
	// On macOS, use osascript
	if runtime.GOOS == "darwin" {
		script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
		exec.Command("osascript", "-e", script).Run()
	}
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}

	cmd.Start()
}

// getTrayIcon returns the icon for the system tray
// Using a simple colored circle icon encoded as PNG
func getTrayIcon() []byte {
	// 16x16 PNG icon - a simple tube/pipe icon
	// This is a minimal PNG with a blue circle
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x10,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0xf3, 0xff, 0x61, 0x00, 0x00, 0x00,
		0x4a, 0x49, 0x44, 0x41, 0x54, 0x38, 0x8d, 0x63, 0x64, 0x60, 0x60, 0xf8,
		0xcf, 0xc0, 0xc0, 0xc0, 0xc4, 0xc0, 0xc0, 0xc0, 0x80, 0x0c, 0x98, 0x18,
		0x18, 0x18, 0xfe, 0x23, 0xc9, 0x33, 0x32, 0xa2, 0x88, 0xb3, 0x30, 0x30,
		0x30, 0xfc, 0x47, 0x92, 0x67, 0x64, 0x44, 0x15, 0x63, 0x65, 0x60, 0x60,
		0xf8, 0x8f, 0x2c, 0xce, 0xc4, 0x80, 0x2c, 0xc6, 0xc2, 0x40, 0x84, 0x0d,
		0x8c, 0xa3, 0x01, 0x00, 0xd7, 0x2e, 0x09, 0xaa, 0x1b, 0x7f, 0xb1, 0xd8,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
