package cmd

import (
	"fmt"
	"sort"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all configured projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList()
	},
}

func runList() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Projects) == 0 {
		fmt.Println("No projects configured.")
		return nil
	}

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	names := make([]string, 0, len(cfg.Projects))
	for name := range cfg.Projects {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		proj := cfg.Projects[name]

		status := green("✓") + " enabled"
		if !proj.Enabled {
			status = red("✗") + " disabled"
		}

		fmt.Printf("%s (%s) %s\n", name, proj.Domain, status)

		svcNames := make([]string, 0, len(proj.Services))
		for svcName := range proj.Services {
			svcNames = append(svcNames, svcName)
		}
		sort.Strings(svcNames)

		for _, svcName := range svcNames {
			svc := proj.Services[svcName]
			fmt.Printf("  %s → %s\n", svcName, svc.Proxy)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(listCmd)
}
