package cloudflare

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func jsonResponse(w http.ResponseWriter, result any) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{
		"success": true,
		"errors":  []any{},
		"result":  result,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func jsonError(w http.ResponseWriter, status int, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resp := map[string]any{
		"success": false,
		"errors":  []map[string]any{{"code": code, "message": msg}},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func TestResolveTunnelToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			resp := map[string]any{"success": false, "errors": []map[string]any{{"code": 9109, "message": "Invalid access token"}}}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		switch {
		case r.URL.Path == "/accounts":
			jsonResponse(w, []account{{ID: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", Name: "My Account"}})
		case strings.HasSuffix(r.URL.Path, "/cfd_tunnel") && r.URL.Query().Get("name") == "my-tunnel":
			jsonResponse(w, []tunnel{{ID: "f1e2d3c4-b5a6-4f1e-ad3c-4b5a6f1e2d3c", Name: "my-tunnel"}})
		case strings.HasSuffix(r.URL.Path, "/f1e2d3c4-b5a6-4f1e-ad3c-4b5a6f1e2d3c/token"):
			jsonResponse(w, "eyJhbGciOiJSUzI1NiJ9.tunnel-jwt-payload")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &Client{
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}

	t.Run("full flow with auto-detected account", func(t *testing.T) {
		token, err := c.ResolveTunnelToken("valid-token", "", "my-tunnel")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "eyJhbGciOiJSUzI1NiJ9.tunnel-jwt-payload" {
			t.Errorf("unexpected token: %q", token)
		}
	})

	t.Run("with explicit account ID", func(t *testing.T) {
		token, err := c.ResolveTunnelToken("valid-token", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", "my-tunnel")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "eyJhbGciOiJSUzI1NiJ9.tunnel-jwt-payload" {
			t.Errorf("unexpected token: %q", token)
		}
	})

	t.Run("empty API token", func(t *testing.T) {
		_, err := c.ResolveTunnelToken("", "", "my-tunnel")
		if err == nil || !strings.Contains(err.Error(), "API token is required") {
			t.Errorf("expected 'API token is required' error, got: %v", err)
		}
	})
}

func TestResolveTunnelToken_InvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := map[string]any{"success": false, "errors": []map[string]any{{"code": 9109, "message": "Invalid access token"}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	_, err := c.ResolveTunnelToken("bad-token", "", "my-tunnel")
	if err == nil || !strings.Contains(err.Error(), "invalid or lacks required permissions") {
		t.Errorf("expected auth error, got: %v", err)
	}
}

func TestResolveTunnelToken_TunnelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/accounts":
			jsonResponse(w, []account{{ID: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", Name: "My Account"}})
		case strings.HasSuffix(r.URL.Path, "/cfd_tunnel"):
			jsonResponse(w, []tunnel{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	_, err := c.ResolveTunnelToken("valid-token", "", "nonexistent")
	if err == nil || !strings.Contains(err.Error(), "not found in account") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestResolveTunnelToken_NoAccounts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, []account{})
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	_, err := c.ResolveTunnelToken("valid-token", "", "my-tunnel")
	if err == nil || !strings.Contains(err.Error(), "no accounts found") {
		t.Errorf("expected 'no accounts' error, got: %v", err)
	}
}

func TestGetTunnelHostnames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			resp := map[string]any{"success": false, "errors": []map[string]any{{"code": 9109, "message": "Invalid access token"}}}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		switch {
		case r.URL.Path == "/accounts":
			jsonResponse(w, []account{{ID: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", Name: "My Account"}})
		case strings.HasSuffix(r.URL.Path, "/cfd_tunnel") && r.URL.Query().Get("name") == "my-tunnel":
			jsonResponse(w, []tunnel{{ID: "f1e2d3c4-b5a6-4f1e-ad3c-4b5a6f1e2d3c", Name: "my-tunnel"}})
		case strings.HasSuffix(r.URL.Path, "/f1e2d3c4-b5a6-4f1e-ad3c-4b5a6f1e2d3c/configurations"):
			jsonResponse(w, map[string]any{
				"config": map[string]any{
					"ingress": []map[string]any{
						{"hostname": "myapp.example.com", "service": "http://localhost:3000"},
						{"hostname": "api.example.com", "service": "http://localhost:8000"},
						{"service": "http_status:404"},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}

	t.Run("extracts hostnames and skips catch-all", func(t *testing.T) {
		hostnames, err := c.GetTunnelHostnames("valid-token", "", "my-tunnel")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(hostnames) != 2 {
			t.Fatalf("expected 2 hostnames, got %d", len(hostnames))
		}
		if hostnames[0] != "myapp.example.com" {
			t.Errorf("expected myapp.example.com, got %s", hostnames[0])
		}
		if hostnames[1] != "api.example.com" {
			t.Errorf("expected api.example.com, got %s", hostnames[1])
		}
	})

	t.Run("with explicit account ID", func(t *testing.T) {
		hostnames, err := c.GetTunnelHostnames("valid-token", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", "my-tunnel")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(hostnames) != 2 {
			t.Fatalf("expected 2 hostnames, got %d", len(hostnames))
		}
	})

	t.Run("empty API token", func(t *testing.T) {
		_, err := c.GetTunnelHostnames("", "", "my-tunnel")
		if err == nil || !strings.Contains(err.Error(), "API token is required") {
			t.Errorf("expected 'API token is required' error, got: %v", err)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := c.GetTunnelHostnames("bad-token", "", "my-tunnel")
		if err == nil || !strings.Contains(err.Error(), "invalid or lacks required permissions") {
			t.Errorf("expected auth error, got: %v", err)
		}
	})
}

func TestGetTunnelHostnames_EmptyIngress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/accounts":
			jsonResponse(w, []account{{ID: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", Name: "My Account"}})
		case strings.HasSuffix(r.URL.Path, "/cfd_tunnel") && r.URL.Query().Get("name") == "empty-tunnel":
			jsonResponse(w, []tunnel{{ID: "f1e2d3c4-b5a6-4f1e-ad3c-4b5a6f1e2d3c", Name: "empty-tunnel"}})
		case strings.HasSuffix(r.URL.Path, "/configurations"):
			jsonResponse(w, map[string]any{
				"config": map[string]any{
					"ingress": []map[string]any{
						{"service": "http_status:404"},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	hostnames, err := c.GetTunnelHostnames("valid-token", "", "empty-tunnel")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hostnames) != 0 {
		t.Errorf("expected 0 hostnames for catch-all-only ingress, got %d", len(hostnames))
	}
}

func TestResolveTunnelToken_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/accounts" {
			jsonResponse(w, []account{{ID: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4", Name: "My Account"}})
			return
		}
		jsonError(w, http.StatusBadRequest, 1000, "something went wrong")
	}))
	defer srv.Close()

	c := &Client{baseURL: srv.URL, httpClient: srv.Client()}
	_, err := c.ResolveTunnelToken("valid-token", "", "my-tunnel")
	if err == nil || !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("expected API error message, got: %v", err)
	}
}
