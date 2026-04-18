package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/httphatch/hatch/internal/api"
)

// DaemonClient wraps HTTP calls to the Hatch daemon REST API.
type DaemonClient struct {
	baseURL string
	http    *http.Client
}

// NewDaemonClient creates a client that talks to the daemon at the default address.
func NewDaemonClient() *DaemonClient {
	return &DaemonClient{
		baseURL: "http://" + api.DefaultAddr,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *DaemonClient) get(path string) (json.RawMessage, error) {
	resp, err := c.http.Get(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("hatch daemon is not running — run 'hatch up' in your terminal to start it")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}
	return body, nil
}

func (c *DaemonClient) postJSON(path string, payload any) (json.RawMessage, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	resp, err := c.http.Post(c.baseURL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("hatch daemon is not running — run 'hatch up' in your terminal to start it")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}
	return body, nil
}

func (c *DaemonClient) delete(path string) (json.RawMessage, error) {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hatch daemon is not running — run 'hatch up' in your terminal to start it")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, body)
	}
	return body, nil
}

// GetStatus returns daemon status.
func (c *DaemonClient) GetStatus() (json.RawMessage, error) {
	return c.get("/api/status")
}

// ListProjects returns all projects.
func (c *DaemonClient) ListProjects() (json.RawMessage, error) {
	return c.get("/api/projects")
}

// GetHealth returns service health statuses.
func (c *DaemonClient) GetHealth() (json.RawMessage, error) {
	return c.get("/api/health")
}

// GetProcesses returns process statuses.
func (c *DaemonClient) GetProcesses() (json.RawMessage, error) {
	return c.get("/api/processes")
}

// CreateSession creates a new session.
func (c *DaemonClient) CreateSession(project, name string, ttl int) (json.RawMessage, error) {
	return c.postJSON("/api/sessions", map[string]any{
		"project": project,
		"name":    name,
		"ttl":     ttl,
	})
}

// ListSessions returns all active sessions.
func (c *DaemonClient) ListSessions() (json.RawMessage, error) {
	return c.get("/api/sessions")
}

// DestroySession destroys a session.
func (c *DaemonClient) DestroySession(project, name string) (json.RawMessage, error) {
	return c.delete(fmt.Sprintf("/api/sessions/%s/%s", url.PathEscape(project), url.PathEscape(name)))
}

// RestartProcess restarts a service process.
func (c *DaemonClient) RestartProcess(project, service string) (json.RawMessage, error) {
	return c.postJSON(fmt.Sprintf("/api/processes/%s/%s/restart", url.PathEscape(project), url.PathEscape(service)), map[string]string{})
}
