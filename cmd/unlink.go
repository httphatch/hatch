package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
)

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Unlink a project from Hatch",
	Long:  `Removes the current project from the central Hatch config.`,
	RunE:  runUnlink,
}

func runUnlink(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		name = filepath.Base(cwd)
	}

	cfg, err := config.LoadRaw()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := config.UnmergeProject(&cfg, name); err != nil {
		return fmt.Errorf("unlink project: %w", err)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Project '%s' unlinked\n", green("✓"), name)

	return nil
}

func init() {
	unlinkCmd.Flags().String("name", "", "override the project name (default: directory basename)")

	rootCmd.AddCommand(unlinkCmd)
}
