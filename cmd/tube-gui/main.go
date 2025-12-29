//go:build darwin

// tube-gui is the graphical interface for tube
// It provides a system tray icon and web dashboard
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/gui"
)

func main() {
	// Load configuration
	configPath := config.ConfigPath()
	cfg, err := config.Load(configPath)
	if err != nil {
		// Try to load with empty config path (will use defaults)
		cfg, err = config.Load("")
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}
	}

	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	// Start dashboard server in background
	dashboard, err := gui.NewDashboard(cfg, configPath)
	if err != nil {
		log.Fatalf("Failed to create dashboard: %v", err)
	}

	go func() {
		fmt.Printf("Dashboard running at http://localhost:%d\n", cfg.Proxy.DashboardPort)
		if err := dashboard.Start(); err != nil {
			log.Printf("Dashboard error: %v", err)
		}
	}()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		dashboard.Stop()
		os.Exit(0)
	}()

	// Run system tray (blocks until quit)
	tray, err := gui.NewTrayApp(cfg, configPath)
	if err != nil {
		log.Fatalf("Failed to create tray app: %v", err)
	}

	tray.Run()
}
