package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage Cloudflare tunnels",
}

var tunnelStartCmd = &cobra.Command{
	Use:   "start <project>/<service>",
	Short: "Start a Cloudflare tunnel for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, service, err := parseTunnelArg(args[0])
		if err != nil {
			return err
		}
		endpoint := fmt.Sprintf("%s/api/tunnels/%s/%s/start", daemonBaseURL(), url.PathEscape(project), url.PathEscape(service))
		client := &http.Client{Timeout: 30 * time.Second}
		httpResp, err := client.Post(endpoint, "application/json", strings.NewReader("{}"))
		if err != nil {
			return fmt.Errorf("daemon not running, start with: hatch up")
		}
		defer func() { _ = httpResp.Body.Close() }()
		if httpResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to start tunnel (status %d)", httpResp.StatusCode)
		}
		var resp struct {
			URL    string `json:"url"`
			Status string `json:"status"`
		}
		if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
		fmt.Printf("%s %s/%s\n", color.New(color.FgGreen, color.Bold).Sprint("Started"), project, service)
		if resp.URL != "" {
			fmt.Println(resp.URL)
		}
		return nil
	},
}

var tunnelStopCmd = &cobra.Command{
	Use:   "stop <project>/<service>",
	Short: "Stop a Cloudflare tunnel for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, service, err := parseTunnelArg(args[0])
		if err != nil {
			return err
		}
		endpoint := fmt.Sprintf("%s/api/tunnels/%s/%s/stop", daemonBaseURL(), url.PathEscape(project), url.PathEscape(service))
		client := &http.Client{Timeout: 30 * time.Second}
		httpResp, err := client.Post(endpoint, "application/json", strings.NewReader("{}"))
		if err != nil {
			return fmt.Errorf("daemon not running, start with: hatch up")
		}
		defer func() { _ = httpResp.Body.Close() }()
		if httpResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to stop tunnel (status %d)", httpResp.StatusCode)
		}
		fmt.Printf("%s %s/%s\n", color.New(color.FgRed, color.Bold).Sprint("Stopped"), project, service)
		return nil
	},
}

var tunnelStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all Cloudflare tunnels",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		endpoint := fmt.Sprintf("%s/api/tunnels", daemonBaseURL())
		client := &http.Client{Timeout: 30 * time.Second}
		httpResp, err := client.Get(endpoint)
		if err != nil {
			return fmt.Errorf("daemon not running, start with: hatch up")
		}
		defer func() { _ = httpResp.Body.Close() }()
		if httpResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get tunnel status (status %d)", httpResp.StatusCode)
		}
		var tunnels []struct {
			Project   string `json:"project"`
			Service   string `json:"service"`
			Running   bool   `json:"running"`
			URL       string `json:"url"`
			Type      string `json:"type"`
			StartedAt string `json:"started_at"`
			Error     string `json:"error"`
		}
		if err := json.NewDecoder(httpResp.Body).Decode(&tunnels); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
		if len(tunnels) == 0 {
			fmt.Println("No active tunnels")
			return nil
		}
		header := color.New(color.Bold)
		_, _ = header.Printf("%-30s %-10s %-50s %-12s\n", "PROJECT/SERVICE", "TYPE", "URL", "UPTIME")
		for _, t := range tunnels {
			name := t.Project + "/" + t.Service
			uptime := ""
			if t.StartedAt != "" {
				if started, err := time.Parse(time.RFC3339, t.StartedAt); err == nil {
					uptime = time.Since(started).Truncate(time.Second).String()
				}
			}
			fmt.Printf("%-30s %-10s %-50s %-12s\n", name, t.Type, t.URL, uptime)
		}
		return nil
	},
}

func parseTunnelArg(arg string) (project, service string, err error) {
	project, service, ok := strings.Cut(arg, "/")
	if !ok || project == "" || service == "" {
		return "", "", fmt.Errorf("invalid format %q, expected <project>/<service>", arg)
	}
	return project, service, nil
}

func init() {
	tunnelCmd.AddCommand(tunnelStartCmd)
	tunnelCmd.AddCommand(tunnelStopCmd)
	tunnelCmd.AddCommand(tunnelStatusCmd)
	rootCmd.AddCommand(tunnelCmd)
}
