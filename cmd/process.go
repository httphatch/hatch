package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/httphatch/hatch/internal/api"
	"github.com/httphatch/hatch/internal/config"
)

// daemonBaseURL returns the HTTP base URL for the daemon API,
// reading the configured api_port from config.yml if set.
func daemonBaseURL() string {
	cfg, err := config.Load()
	if err != nil || cfg.Settings.APIPort == 0 {
		return "http://" + api.DefaultAddr
	}
	return fmt.Sprintf("http://127.0.0.1:%d", cfg.Settings.APIPort)
}

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Manage individual processes",
}

var processStopCmd = &cobra.Command{
	Use:   "stop <project>/<service>",
	Short: "Stop a managed process",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, service, err := parseProcessArg(args[0])
		if err != nil {
			return err
		}
		if err := postProcessAction(project, service, "stop"); err != nil {
			return err
		}
		fmt.Printf("%s %s/%s\n", color.New(color.FgRed, color.Bold).Sprint("Stopped"), project, service)
		return nil
	},
}

var processStartCmd = &cobra.Command{
	Use:   "start <project>/<service>",
	Short: "Start a stopped process",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, service, err := parseProcessArg(args[0])
		if err != nil {
			return err
		}
		if err := postProcessAction(project, service, "start"); err != nil {
			return err
		}
		fmt.Printf("%s %s/%s\n", color.New(color.FgGreen, color.Bold).Sprint("Started"), project, service)
		return nil
	},
}

var processRestartCmd = &cobra.Command{
	Use:   "restart <project>/<service>",
	Short: "Restart a managed process",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, service, err := parseProcessArg(args[0])
		if err != nil {
			return err
		}
		if err := postProcessAction(project, service, "restart"); err != nil {
			return err
		}
		fmt.Printf("%s %s/%s\n", color.New(color.FgCyan, color.Bold).Sprint("Restarted"), project, service)
		return nil
	},
}

func parseProcessArg(arg string) (project, service string, err error) {
	project, service, ok := strings.Cut(arg, "/")
	if !ok || project == "" || service == "" {
		return "", "", fmt.Errorf("invalid format %q, expected <project>/<service>", arg)
	}
	return project, service, nil
}

func postProcessAction(project, service, action string) error {
	endpoint := fmt.Sprintf("%s/api/processes/%s/%s/%s", daemonBaseURL(), url.PathEscape(project), url.PathEscape(service), action)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(endpoint, "application/json", strings.NewReader("{}"))
	if err != nil {
		return fmt.Errorf("daemon not running, start with: hatch up")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to %s process (status %d)", action, resp.StatusCode)
	}
	return nil
}

func init() {
	processCmd.AddCommand(processStopCmd)
	processCmd.AddCommand(processStartCmd)
	processCmd.AddCommand(processRestartCmd)
	rootCmd.AddCommand(processCmd)
}
