package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
)

var enableCmd = &cobra.Command{
	Use:               "enable <project>",
	Short:             "Enable a project",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeProjectNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		return setProjectEnabled(args[0], true)
	},
}

func setProjectEnabled(name string, enabled bool) error {
	cfg, err := config.LoadRaw()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	proj, exists := cfg.Projects[name]
	if !exists {
		return fmt.Errorf("project %q not found", name)
	}

	action := "enabled"
	if !enabled {
		action = "disabled"
	}

	if proj.Enabled == enabled {
		fmt.Printf("Project '%s' is already %s\n", name, action)
		return nil
	}

	proj.Enabled = enabled
	cfg.Projects[name] = proj

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Project '%s' %s\n", green("✓"), name, action)
	return nil
}

func init() {
	rootCmd.AddCommand(enableCmd)
}
