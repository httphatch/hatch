package daemon

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/httphatch/hatch/internal/config"
)

// PlistLabel is the launchd job label for the Hatch daemon.
const PlistLabel = "dev.hatch.daemon"

// LaunchdConfig holds the configuration for generating a launchd plist.
type LaunchdConfig struct {
	Label            string
	BinaryPath       string
	WorkingDirectory string
	LogDir           string
	KeepAlive        bool
}

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{ .Label }}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{ .BinaryPath }}</string>
		<string>_run</string>
	</array>
	<key>WorkingDirectory</key>
	<string>{{ .WorkingDirectory }}</string>
	<key>RunAtLoad</key>
	{{ if .KeepAlive }}<true/>{{ else }}<false/>{{ end }}
	<key>KeepAlive</key>
	{{ if .KeepAlive }}<true/>{{ else }}<false/>{{ end }}
	<key>StandardOutPath</key>
	<string>{{ .LogDir }}/daemon.log</string>
	<key>StandardErrorPath</key>
	<string>{{ .LogDir }}/daemon.log</string>
</dict>
</plist>
`

// PlistPath returns the path to the launchd plist file.
func PlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", PlistLabel+".plist"), nil
}

// GeneratePlist renders the plist XML from the given config.
func GeneratePlist(cfg LaunchdConfig) ([]byte, error) {
	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse plist template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return nil, fmt.Errorf("execute plist template: %w", err)
	}
	return buf.Bytes(), nil
}

// InstallPlist generates and writes the plist file to ~/Library/LaunchAgents/.
func InstallPlist(cfg LaunchdConfig) error {
	data, err := GeneratePlist(cfg)
	if err != nil {
		return err
	}

	path, err := PlistPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}
	return nil
}

// UninstallPlist removes the plist file.
func UninstallPlist() error {
	path, err := PlistPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	return nil
}

// LoadPlist loads the plist into launchd using launchctl.
func LoadPlist() error {
	path, err := PlistPath()
	if err != nil {
		return err
	}
	out, err := exec.Command("launchctl", "load", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// UnloadPlist unloads the plist from launchd using launchctl.
func UnloadPlist() error {
	path, err := PlistPath()
	if err != nil {
		return err
	}
	out, err := exec.Command("launchctl", "unload", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl unload: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// IsLoaded checks whether the launchd job is currently loaded.
func IsLoaded() bool {
	err := exec.Command("launchctl", "list", PlistLabel).Run()
	return err == nil
}

// DefaultLaunchdConfig returns a LaunchdConfig with sensible defaults.
func DefaultLaunchdConfig(autoStart bool) (LaunchdConfig, error) {
	bin, err := os.Executable()
	if err != nil {
		return LaunchdConfig{}, fmt.Errorf("get executable path: %w", err)
	}

	return LaunchdConfig{
		Label:            PlistLabel,
		BinaryPath:       bin,
		WorkingDirectory: config.Dir(),
		LogDir:           config.LogsDir(),
		KeepAlive:        autoStart,
	}, nil
}
