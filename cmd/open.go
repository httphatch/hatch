package cmd

import (
	"fmt"
	"os/exec"
	"sort"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
)

var openCmd = &cobra.Command{
	Use:               "open [project]",
	Short:             "Open a project in the browser",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE:              runOpen,
}

func runOpen(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadWithProjectConfigs()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Projects) == 0 {
		return fmt.Errorf("no projects configured")
	}

	var name string
	switch {
	case len(args) == 1:
		name = args[0]
	case len(cfg.Projects) == 1:
		for n := range cfg.Projects {
			name = n
		}
	default:
		fmt.Println("Multiple projects configured. Specify one:")
		names := make([]string, 0, len(cfg.Projects))
		for n := range cfg.Projects {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Printf("  hatch open %s\n", n)
		}
		return nil
	}

	proj, exists := cfg.Projects[name]
	if !exists {
		return fmt.Errorf("project %q not found", name)
	}

	url := "https://" + proj.Domain
	if cfg.Settings.HTTPSPort != 443 {
		url = fmt.Sprintf("%s:%d", url, cfg.Settings.HTTPSPort)
	}

	if err := exec.Command("open", url).Run(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Opened %s\n", green("✓"), url)
	return nil
}

func init() {
	rootCmd.AddCommand(openCmd)
}
