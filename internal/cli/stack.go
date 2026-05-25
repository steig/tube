package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/steig/tube/internal/config"
	"github.com/steig/tube/internal/proxy"
	"github.com/steig/tube/internal/service"
)

// cmdStack is the standard bundle of objects most commands need.
// configPath is the resolved file path we should write to (never empty).
type cmdStack struct {
	cfg        *config.Config
	configPath string
	pm         *service.ProcessManager
	ngx        *proxy.NginxManager
	dms        *proxy.DnsmasqManager
}

// resolveConfigPath returns the user's --config flag or the default ~/.tube/config.yaml.
// We always want a concrete path so commands that save can honor --config (see SSL bug).
func resolveConfigPath(cmd *cobra.Command) string {
	if p, _ := cmd.Flags().GetString("config"); p != "" {
		return p
	}
	return config.ConfigPath()
}

// loadCfg loads the configuration. Used by commands that don't need the
// full proxy/service stack (config, list, etc).
func loadCfg(cmd *cobra.Command) (*config.Config, string, error) {
	configPath := resolveConfigPath(cmd)
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, configPath, fmt.Errorf("failed to load configuration: %w", err)
	}
	return cfg, configPath, nil
}

// loadStack loads the config and wires up the full proxy + service stack.
// Creates the pid and nginx config directories as a side effect.
func loadStack(cmd *cobra.Command) (*cmdStack, error) {
	cfg, configPath, err := loadCfg(cmd)
	if err != nil {
		return nil, err
	}

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

	return &cmdStack{cfg: cfg, configPath: configPath, pm: pm, ngx: ngx, dms: dms}, nil
}
