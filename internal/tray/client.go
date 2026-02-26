//go:build darwin

package tray

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/httphatch/hatch/internal/api"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/daemon"
	"github.com/httphatch/hatch/internal/health"
)

// Client communicates with the daemon over its HTTP API and manages
// the daemon lifecycle via launchctl.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Client that talks to the daemon at the default API address.
func NewClient() *Client {
	return &Client{
		baseURL: "http://" + api.DefaultAddr,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// DaemonStatus checks whether the daemon is reachable and returns its version.
func (c *Client) DaemonStatus() (running bool, version string, err error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/status")
	if err != nil {
		return false, "", nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return false, "", nil
	}

	var status struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return true, "", nil
	}
	return true, status.Version, nil
}

// Health fetches service health statuses from the daemon API.
func (c *Client) Health() (map[health.ServiceKey]health.ServiceStatus, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/health")
	if err != nil {
		return nil, fmt.Errorf("health request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health request: status %d", resp.StatusCode)
	}

	var items []struct {
		Project   string `json:"project"`
		Service   string `json:"service"`
		Status    string `json:"status"`
		Addr      string `json:"addr"`
		Since     string `json:"since"`
		LastCheck string `json:"last_check"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode health: %w", err)
	}

	result := make(map[health.ServiceKey]health.ServiceStatus, len(items))
	for _, item := range items {
		key := health.ServiceKey{Project: item.Project, Service: item.Service}

		var st health.Status
		switch item.Status {
		case "healthy":
			st = health.StatusHealthy
		case "unhealthy":
			st = health.StatusUnhealthy
		default:
			st = health.StatusUnknown
		}

		since, _ := time.Parse(time.RFC3339, item.Since)
		lastCheck, _ := time.Parse(time.RFC3339, item.LastCheck)

		result[key] = health.ServiceStatus{
			Status:    st,
			Addr:      item.Addr,
			Since:     since,
			LastCheck: lastCheck,
		}
	}
	return result, nil
}

// ReloadConfig tells the daemon to reload its configuration.
func (c *Client) ReloadConfig() error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/restart", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("reload request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reload: status %d", resp.StatusCode)
	}
	return nil
}

// StartDaemon installs and loads the launchd plist to start the daemon.
func (c *Client) StartDaemon() error {
	cfg, err := config.LoadWithProjectConfigs()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	launchdCfg, err := daemon.DefaultLaunchdConfig(cfg.Settings.AutoStart)
	if err != nil {
		return fmt.Errorf("launchd config: %w", err)
	}

	if err := daemon.InstallPlist(launchdCfg); err != nil {
		return fmt.Errorf("install plist: %w", err)
	}

	if err := daemon.LoadPlist(); err != nil {
		return fmt.Errorf("load plist: %w", err)
	}
	return nil
}

// StopDaemon uninstalls the plist (to prevent KeepAlive restart) then removes
// the launchd job by label.
func (c *Client) StopDaemon() error {
	if err := daemon.UninstallPlist(); err != nil {
		return fmt.Errorf("uninstall plist: %w", err)
	}
	if err := daemon.RemoveJob(); err != nil {
		return fmt.Errorf("remove job: %w", err)
	}
	return nil
}

// RestartDaemon stops then starts the daemon. It spawns a detached shell
// process for the unload+load sequence so the operation completes even if
// the current daemon process exits.
func (c *Client) RestartDaemon() error {
	path, err := daemon.PlistPath()
	if err != nil {
		return fmt.Errorf("plist path: %w", err)
	}
	cmd := exec.Command("sh", "-c",
		`launchctl unload "$PLIST_PATH" && launchctl load "$PLIST_PATH"`)
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + os.Getenv("HOME"),
		"PLIST_PATH=" + path,
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("restart: spawn failed: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	log.Debug().Msg("tray: restart shell spawned")
	return nil
}
