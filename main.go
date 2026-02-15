package main

import (
	"os"

	"github.com/httphatch/hatch/cmd"
)

func main() {
	cmd.SetAssets(assets)
	cmd.SetAppIcon(appIcon)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
