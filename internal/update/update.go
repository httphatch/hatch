package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const releasesURL = "https://api.github.com/repos/httphatch/hatch/releases/latest"

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// CheckLatest fetches the latest release from GitHub. The caller may set a
// timeout on the provided http.Client (e.g. 3 s for a background check).
func CheckLatest(client *http.Client) (*Release, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	req, err := http.NewRequest("GET", releasesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "hatch-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api: status %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &rel, nil
}

// AssetForArch returns the asset matching the current darwin architecture.
func AssetForArch(assets []Asset) (*Asset, error) {
	target := fmt.Sprintf("hatch-darwin-%s", runtime.GOARCH)
	for _, a := range assets {
		if a.Name == target {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("no asset found for %s", target)
}

var allowedDownloadHosts = map[string]bool{
	"github.com":                    true,
	"objects.githubusercontent.com": true,
}

// Download fetches downloadURL into a temporary file in destDir,
// returning the temp file path.
func Download(client *http.Client, downloadURL, destDir string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Minute}
	}

	parsed, err := url.Parse(downloadURL)
	if err != nil {
		return "", fmt.Errorf("parse download url: %w", err)
	}
	if !allowedDownloadHosts[parsed.Hostname()] {
		return "", fmt.Errorf("untrusted download host: %s", parsed.Hostname())
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("download binary: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: status %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp(destDir, "hatch-update-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("write binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", fmt.Errorf("close temp file: %w", err)
	}

	return tmp.Name(), nil
}

// Apply replaces the binary at binaryPath with the new binary at newPath.
// Uses an atomic os.Rename to avoid leaving the user without a binary.
func Apply(binaryPath, newPath string) error {
	if err := os.Chmod(newPath, 0o755); err != nil {
		return fmt.Errorf("chmod new binary: %w", err)
	}

	if err := os.Rename(newPath, binaryPath); err != nil {
		return fmt.Errorf("install new binary: %w", err)
	}

	return nil
}

// IsHomebrew returns true if the binary path looks like a Homebrew install.
func IsHomebrew(execPath string) bool {
	dir := filepath.Clean(execPath)
	for dir != "/" && dir != "." {
		base := filepath.Base(dir)
		if base == "Cellar" || base == "homebrew" {
			return true
		}
		dir = filepath.Dir(dir)
	}
	return false
}
