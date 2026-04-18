package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage ephemeral project sessions",
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active sessions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(daemonBaseURL() +"/api/sessions")
		if err != nil {
			return fmt.Errorf("daemon not running, start with: hatch up")
		}
		defer func() { _ = resp.Body.Close() }()

		var sessions []struct {
			Project    string            `json:"project"`
			Name       string            `json:"name"`
			State      string            `json:"state"`
			Domains    map[string]string `json:"domains"`
			Ports      map[string]int    `json:"ports"`
			LastActive string            `json:"last_active"`
			TTL        int               `json:"ttl"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No active sessions")
			return nil
		}

		bold := color.New(color.Bold)
		for _, s := range sessions {
			_, _ = bold.Printf("%s/%s", s.Project, s.Name)
			fmt.Printf("  [%s]\n", s.State)
			for svc, domain := range s.Domains {
				fmt.Printf("  %s: https://%s (port %d)\n", svc, domain, s.Ports[svc])
			}
			fmt.Printf("  ttl: %ds\n", s.TTL)
		}
		return nil
	},
}

var sessionStopCmd = &cobra.Command{
	Use:   "stop <project>/<name>",
	Short: "Stop a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, name, err := parseProcessArg(args[0])
		if err != nil {
			return err
		}

		client := &http.Client{Timeout: 30 * time.Second}
		endpoint := fmt.Sprintf("%s/api/sessions/%s/%s", daemonBaseURL(), url.PathEscape(project), url.PathEscape(name))
		req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("daemon not running, start with: hatch up")
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("failed to stop session (status %d): %s", resp.StatusCode, body)
		}
		fmt.Printf("%s %s/%s\n", color.New(color.FgRed, color.Bold).Sprint("Stopped"), project, name)
		return nil
	},
}

var sessionStopAllCmd = &cobra.Command{
	Use:   "stop-all",
	Short: "Stop all active sessions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(daemonBaseURL() +"/api/sessions")
		if err != nil {
			return fmt.Errorf("daemon not running, start with: hatch up")
		}
		defer func() { _ = resp.Body.Close() }()

		var sessions []struct {
			Project string `json:"project"`
			Name    string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No active sessions")
			return nil
		}

		for _, s := range sessions {
			endpoint := fmt.Sprintf("%s/api/sessions/%s/%s", daemonBaseURL(), url.PathEscape(s.Project), url.PathEscape(s.Name))
			req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
			if err != nil {
				continue
			}
			r, err := client.Do(req)
			if err != nil {
				fmt.Printf("  failed to stop %s/%s: %v\n", s.Project, s.Name, err)
				continue
			}
			_ = r.Body.Close()
			fmt.Printf("%s %s/%s\n", color.New(color.FgRed, color.Bold).Sprint("Stopped"), s.Project, s.Name)
		}
		return nil
	},
}

func init() {
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionStopCmd)
	sessionCmd.AddCommand(sessionStopAllCmd)
	rootCmd.AddCommand(sessionCmd)
}
