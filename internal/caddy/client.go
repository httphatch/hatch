package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPClient is the interface for making HTTP requests, allowing injection
// for testing.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client communicates with the Caddy admin API.
type Client struct {
	AdminAddr  string
	HTTPClient HTTPClient
}

// NewClient returns a Client configured with the default admin address.
func NewClient() *Client {
	return &Client{
		AdminAddr:  DefaultAdminAddr,
		HTTPClient: &http.Client{},
	}
}

// Load pushes the given Caddy JSON configuration via POST /load.
// It returns an error if marshaling fails, the request fails, or Caddy
// responds with a non-200 status.
func (c *Client) Load(ctx context.Context, caddyConfig map[string]any) error {
	body, err := json.Marshal(caddyConfig)
	if err != nil {
		return fmt.Errorf("marshaling caddy config: %w", err)
	}

	reqURL := fmt.Sprintf("http://%s/load", c.AdminAddr)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting caddy config: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024))
		return fmt.Errorf("caddy rejected config (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
