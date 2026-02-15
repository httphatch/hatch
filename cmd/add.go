package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
)

var addCmd = &cobra.Command{
	Use:   "add <project>",
	Short: "Add or update a project",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.LoadRaw()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	domain, _ := cmd.Flags().GetString("domain")
	proxy, _ := cmd.Flags().GetString("proxy")
	path, _ := cmd.Flags().GetString("path")

	if domain == "" {
		domain = name + "." + cfg.Settings.TLD
	}

	_, existed := cfg.Projects[name]

	pc := config.ProjectConfig{
		Domain: domain,
		Services: map[string]config.Service{
			"web": {Proxy: proxy},
		},
	}

	if err := config.MergeProjectConfig(&cfg, name, path, pc); err != nil {
		return fmt.Errorf("add project: %w", err)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	if existed {
		fmt.Printf("%s Project '%s' updated (%s)\n", green("✓"), name, domain)
	} else {
		fmt.Printf("%s Project '%s' added (%s)\n", green("✓"), name, domain)
	}

	return nil
}

func init() {
	cwd, _ := os.Getwd()

	addCmd.Flags().String("domain", "", "domain for the project (default: <name>.<tld>)")
	addCmd.Flags().String("proxy", "http://localhost:3000", "upstream proxy target")
	addCmd.Flags().String("path", cwd, "project directory path")

	rootCmd.AddCommand(addCmd)
}
