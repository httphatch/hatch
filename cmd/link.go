package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
)

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link a project from its hatch.yml",
	Long:  `Reads hatch.yml from the current directory and merges the project into the central Hatch config.`,
	RunE:  runLink,
}

func runLink(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	hatchFile := filepath.Join(cwd, "hatch.yml")
	if _, err := os.Stat(hatchFile); err != nil {
		return fmt.Errorf("no hatch.yml found in current directory")
	}

	pc, err := config.LoadProjectConfig(hatchFile)
	if err != nil {
		return fmt.Errorf("load project config: %w", err)
	}

	cfg, err := config.LoadRaw()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		name = filepath.Base(cwd)
	}

	_, existed := cfg.Projects[name]

	if err := config.MergeProjectConfig(&cfg, name, cwd, pc); err != nil {
		return fmt.Errorf("link project: %w", err)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	if existed {
		fmt.Printf("%s Project '%s' updated (%s)\n", green("✓"), name, pc.Domain)
	} else {
		fmt.Printf("%s Project '%s' linked (%s)\n", green("✓"), name, pc.Domain)
	}

	return nil
}

func init() {
	linkCmd.Flags().String("name", "", "override the project name (default: directory basename)")

	rootCmd.AddCommand(linkCmd)
}
