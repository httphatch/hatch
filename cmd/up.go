package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/api"
	"github.com/httphatch/hatch/internal/certs"
	"github.com/httphatch/hatch/internal/config"
	"github.com/httphatch/hatch/internal/daemon"
	"github.com/httphatch/hatch/internal/dns"
)

const daemonStartTimeout = 120 * time.Second

var upCmd = &cobra.Command{
	Use:     "up",
	Aliases: []string{"start"},
	Short:   "Start the Hatch daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUp()
	},
}

func runUp() error {
	// Check if already running (idempotent).
	running, pid, err := daemon.IsRunning()
	if err != nil {
		return fmt.Errorf("check running: %w", err)
	}
	if running {
		fmt.Printf("%s already running (pid %d)\n", color.CyanString("Hatch"), pid)
		return nil
	}

	// Initialize config directory and default config file.
	if err := config.Init(); err != nil {
		return fmt.Errorf("init config: %w", err)
	}
	log.Debug().Msg("config initialized")

	// Load config to read settings.
	cfg, err := config.LoadWithProjectConfigs()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Generate CA if needed.
	caPaths := certs.NewCAPaths(config.CertsDir())
	if !certs.CAExists(caPaths) {
		log.Info().Msg("generating root CA")
		if err := certs.GenerateCA(caPaths); err != nil {
			return fmt.Errorf("generate CA: %w", err)
		}
	}

	// Generate intermediate CA if needed.
	if !certs.IntermediateCAExists(caPaths) {
		log.Info().Msg("generating intermediate CA")
		if err := certs.GenerateIntermediateCA(caPaths); err != nil {
			return fmt.Errorf("generate intermediate CA: %w", err)
		}
	}

	// Trust CA if needed.
	if !certs.IsCATrusted(caPaths.Cert) {
		log.Info().Msg("trusting root CA (may prompt for password)")
		if err := certs.TrustCA(&sudoRunner{}, caPaths.Cert); err != nil {
			return fmt.Errorf("trust CA: %w", err)
		}
	}

	// Install DNS resolver if needed.
	if !dns.IsResolverInstalled(cfg.Settings.TLD) {
		log.Info().Str("tld", cfg.Settings.TLD).Msg("installing DNS resolver (may prompt for password)")
		if err := dns.InstallResolverFile(&sudoRunner{}, cfg.Settings.TLD, dns.DefaultListenIP, dns.DefaultPort); err != nil {
			return fmt.Errorf("install resolver: %w", err)
		}
	}

	// Pre-flight: catch port conflicts before loading the plist so the error
	// is visible in the terminal. The daemon performs the same check at startup.
	for _, port := range []int{cfg.Settings.HTTPPort, cfg.Settings.HTTPSPort} {
		info, err := daemon.CheckPort(port)
		if err != nil {
			log.Warn().Err(err).Int("port", port).Msg("could not check port availability")
		} else if info != nil {
			return fmt.Errorf("port %d is already in use by %s; stop that process first", port, info)
		}
	}

	// Build launchd config and install plist.
	launchdCfg, err := daemon.DefaultLaunchdConfig(cfg.Settings.AutoStart)
	if err != nil {
		return fmt.Errorf("launchd config: %w", err)
	}

	if err := daemon.InstallPlist(launchdCfg); err != nil {
		return fmt.Errorf("install plist: %w", err)
	}
	log.Debug().Msg("plist installed")

	// Load the plist into launchd.
	if err := daemon.LoadPlist(); err != nil {
		return fmt.Errorf("load plist: %w", err)
	}

	// Verify the daemon actually started by polling the API.
	if err := waitForDaemon(); err != nil {
		return err
	}

	// Launch the tray app as a separate process if enabled.
	if cfg.Settings.TrayIcon {
		launchTray(false)
	}

	fmt.Printf("%s started\n", color.New(color.FgCyan, color.Bold).Sprint("Hatch"))
	return nil
}

// waitForDaemon polls the daemon API while tailing the log file for progress.
// It returns when the API responds, or fails if the daemon process exits.
func waitForDaemon() error {
	ctx, cancel := context.WithTimeout(context.Background(), daemonStartTimeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	statusURL := "http://" + api.DefaultAddr + "/api/status"

	// Open the log file for tailing progress.
	logPath := config.LogFile()
	logFile, err := os.Open(logPath)
	if err != nil {
		logFile = nil
	} else {
		_, _ = logFile.Seek(0, 2)
		defer logFile.Close() //nolint:errcheck // best-effort close on read-only file
	}

	dim := color.New(color.Faint)
	lastProgress := time.Now()
	readBuf := make([]byte, 4096)
	var lineBuf []byte

	for {
		// Read new log data and show progress lines.
		if logFile != nil {
			for {
				n, _ := logFile.Read(readBuf)
				if n == 0 {
					break
				}
				lineBuf = append(lineBuf, readBuf[:n]...)
				for {
					idx := bytes.IndexByte(lineBuf, '\n')
					if idx < 0 {
						break
					}
					line := string(lineBuf[:idx])
					lineBuf = lineBuf[idx+1:]
					if label := startupLabel(line); label != "" {
						_, _ = dim.Fprintf(os.Stderr, "  %s\n", label)
						lastProgress = time.Now()
					}
				}
			}
		}

		// Periodic nudge if no progress has been shown for a while.
		if time.Since(lastProgress) > 10*time.Second {
			_, _ = dim.Fprintf(os.Stderr, "  Waiting for daemon...\n")
			lastProgress = time.Now()
		}

		// Check if the API is up.
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
		if reqErr != nil {
			return fmt.Errorf("creating request: %w", reqErr)
		}
		resp, respErr := client.Do(req)
		if respErr == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		// Check if the daemon process died.
		alive, _, _ := daemon.IsRunning()
		if !alive {
			// Give it a moment; the PID file may not be written yet.
			if time.Since(lastProgress) > 5*time.Second {
				return fmt.Errorf("daemon process exited; check logs with: hatch logs")
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for daemon after 120s; check logs with: hatch logs")
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// startupLabel maps a daemon log message to a user-friendly progress label.
func startupLabel(line string) string {
	var entry struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return ""
	}
	msg := entry.Message

	labels := []struct {
		match string
		label string
	}{
		{"dns server started", "Starting DNS server"},
		{"caddy server started", "Starting Caddy server"},
		{"resolved tunnel domains", "Resolving tunnel domains"},
		{"caddy config loaded", "Loading Caddy config"},
		{"health checker started", "Starting health checker"},
		{"process manager started", "Starting process manager"},
		{"tunnel manager started", "Starting tunnel manager"},
		{"api server started", "Starting API server"},
		{"config watcher started", "Starting config watcher"},
		{"daemon running", "Ready"},
	}

	for _, l := range labels {
		if strings.Contains(msg, l.match) {
			return l.label
		}
	}
	return ""
}

func init() {
	rootCmd.AddCommand(upCmd)
}
