package cmd

import (
	"sort"

	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/config"
)

// completeProjectNames returns a shell-completion function that suggests
// configured project names.
func completeProjectNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.LoadRaw()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names := make([]string, 0, len(cfg.Projects))
	for name := range cfg.Projects {
		names = append(names, name)
	}
	sort.Strings(names)

	return names, cobra.ShellCompDirectiveNoFileComp
}
