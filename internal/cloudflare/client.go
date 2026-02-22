package cloudflare

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

var validAccountID = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
var validUUID = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)

const defaultBaseURL = "https://api.cloudflare.com/client/v4"

type apiResponse struct {
	Success bool            `json:"success"`
	Errors  []apiError      `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type tunnel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Client calls the Cloudflare API to resolve tunnel tokens.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Client with the default Cloudflare API base URL.
func NewClient() *Client {
	return &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ResolveTunnelToken resolves a tunnel JWT from a Cloudflare API token and tunnel name.
// If accountID is empty, it auto-detects the account from the API token.
func (c *Client) ResolveTunnelToken(apiToken, accountID, tunnelName string) (string, error) {
	if apiToken == "" {
		return "", fmt.Errorf("cloudflare API token is required")
	}

	var err error
	if accountID == "" {
		accountID, err = c.getAccountID(apiToken)
		if err != nil {
			return "", fmt.Errorf("looking up cloudflare account: %w", err)
		}
	}

	tunnelID, err := c.findTunnelByName(apiToken, accountID, tunnelName)
	if err != nil {
		return "", err
	}

	token, err := c.getTunnelToken(apiToken, accountID, tunnelID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (c *Client) getAccountID(apiToken string) (string, error) {
	body, err := c.doGet(apiToken, c.baseURL+"/accounts")
	if err != nil {
		return "", err
	}

	var accounts []account
	if err := json.Unmarshal(body, &accounts); err != nil {
		return "", fmt.Errorf("parsing accounts response: %w", err)
	}
	if len(accounts) == 0 {
		return "", fmt.Errorf("no accounts found for this API token")
	}

	id := accounts[0].ID
	if !validAccountID.MatchString(id) {
		return "", fmt.Errorf("unexpected account ID format from Cloudflare API: %q", id)
	}
	return id, nil
}

func (c *Client) findTunnelByName(apiToken, accountID, tunnelName string) (string, error) {
	u := fmt.Sprintf("%s/accounts/%s/cfd_tunnel?name=%s&is_deleted=false",
		c.baseURL, url.PathEscape(accountID), url.QueryEscape(tunnelName))

	body, err := c.doGet(apiToken, u)
	if err != nil {
		return "", err
	}

	var tunnels []tunnel
	if err := json.Unmarshal(body, &tunnels); err != nil {
		return "", fmt.Errorf("parsing tunnels response: %w", err)
	}
	if len(tunnels) == 0 {
		return "", fmt.Errorf("tunnel %q not found in account", tunnelName)
	}

	id := tunnels[0].ID
	if !validUUID.MatchString(id) {
		return "", fmt.Errorf("unexpected tunnel ID format from Cloudflare API: %q", id)
	}
	return id, nil
}

func (c *Client) getTunnelToken(apiToken, accountID, tunnelID string) (string, error) {
	u := fmt.Sprintf("%s/accounts/%s/cfd_tunnel/%s/token",
		c.baseURL, url.PathEscape(accountID), url.PathEscape(tunnelID))

	body, err := c.doGet(apiToken, u)
	if err != nil {
		return "", err
	}

	var token string
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("parsing tunnel token response: %w", err)
	}
	if token == "" {
		return "", fmt.Errorf("cloudflare returned an empty tunnel token")
	}

	return token, nil
}

func (c *Client) doGet(apiToken, rawURL string) (json.RawMessage, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading cloudflare API response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("cloudflare API token is invalid or lacks required permissions (HTTP %d)", resp.StatusCode)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing cloudflare API response (HTTP %d): %w", resp.StatusCode, err)
	}

	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("cloudflare API error: %s (code %d)", apiResp.Errors[0].Message, apiResp.Errors[0].Code)
		}
		return nil, fmt.Errorf("cloudflare API request failed (HTTP %d)", resp.StatusCode)
	}

	return apiResp.Result, nil
}
