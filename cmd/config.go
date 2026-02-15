package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open the Hatch config file in your editor",
	Long:  `Opens the Hatch config file in $EDITOR (falls back to $VISUAL, then vi).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfig()
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the Hatch config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigValidate()
	},
}

func runConfig() error {
	path := config.ConfigFile()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("config file not found at %s — run %s first", path, color.New(color.Bold).Sprint("hatch init"))
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}

	args := strings.Fields(editor)
	args = append(args, path)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with error: %w", err)
	}

	return nil
}

func runConfigValidate() error {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	cfg, err := config.LoadRaw()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	errs := config.Validate(cfg)
	if len(errs) == 0 {
		fmt.Printf("%s Config is valid\n", green("✓"))
		return nil
	}

	fmt.Printf("%s Config has %d error(s):\n", red("✗"), len(errs))
	for _, e := range errs {
		fmt.Printf("  - %s\n", e)
	}

	return fmt.Errorf("config validation failed")
}

func init() {
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}
