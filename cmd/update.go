package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/daemon"
	"github.com/httphatch/hatch/internal/update"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Hatch to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func runUpdate() error {
	yellow := color.New(color.FgYellow).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	// Parse current version.
	currentRaw := strings.TrimPrefix(version, "v")
	current, err := semver.NewVersion(currentRaw)
	if err != nil {
		fmt.Printf("%s Running a dev build (%s) — cannot self-update.\n", yellow("!"), version)
		fmt.Println("  Download the latest release from:")
		fmt.Println("  https://github.com/httphatch/hatch/releases")
		return nil
	}

	// Check for Homebrew install.
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("detect executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	if update.IsHomebrew(execPath) {
		fmt.Printf("%s Installed via Homebrew. Run: %s\n", yellow("!"), color.CyanString("brew upgrade hatch"))
		return nil
	}

	// Fetch latest release.
	fmt.Println("Checking for updates...")
	rel, err := update.CheckLatest(nil)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}

	latestRaw := strings.TrimPrefix(rel.TagName, "v")
	latest, err := semver.NewVersion(latestRaw)
	if err != nil {
		return fmt.Errorf("parse latest version %q: %w", rel.TagName, err)
	}

	if !current.LessThan(latest) {
		fmt.Printf("%s Hatch %s is already up to date.\n", green("✓"), version)
		return nil
	}

	// Find matching asset.
	asset, err := update.AssetForArch(rel.Assets)
	if err != nil {
		return fmt.Errorf("find download: %w", err)
	}

	// Download new binary.
	fmt.Printf("Downloading %s...\n", rel.TagName)
	tmpPath, err := update.Download(nil, asset.BrowserDownloadURL, filepath.Dir(execPath))
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	// Check if daemon is running so we can restart after update.
	wasRunning, _, _ := daemon.IsRunning()

	if wasRunning {
		fmt.Println("Stopping daemon...")
		if err := runDown(); err != nil {
			return fmt.Errorf("stop daemon: %w", err)
		}
	}

	// Apply update (atomic replace).
	if err := update.Apply(execPath, tmpPath); err != nil {
		return fmt.Errorf("apply update: %w", err)
	}
	cleanup = false

	if wasRunning {
		fmt.Println("Restarting daemon...")
		if err := runUp(); err != nil {
			return fmt.Errorf("restart daemon: %w", err)
		}
	}

	fmt.Printf("\n%s Updated Hatch %s → %s\n",
		green("✓"),
		color.New(color.FgWhite).Sprintf("v%s", current),
		color.New(color.FgGreen).Sprintf("v%s", latest),
	)
	return nil
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
