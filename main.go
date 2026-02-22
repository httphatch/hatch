package main

import (
	"fmt"
	"os"

	"github.com/httphatch/hatch/cmd"
)

func main() {
	cmd.SetAssets(assets)
	cmd.SetAppIcon(appIcon)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
