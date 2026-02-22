package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/api"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Open the Hatch dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Post("http://"+api.DefaultAddr+"/api/dashboard/show", "application/json", nil)
		if err != nil {
			return fmt.Errorf("daemon not running — start with: hatch up")
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to show dashboard (status %d)", resp.StatusCode)
		}
		fmt.Println("Dashboard opened.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(appCmd)
}
