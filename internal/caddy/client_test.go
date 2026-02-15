package caddy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_Load_Success(t *testing.T) {
	var gotContentType string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &Client{
		AdminAddr:  strings.TrimPrefix(srv.URL, "http://"),
		HTTPClient: srv.Client(),
	}

	cfg := map[string]any{"admin": map[string]any{"listen": "localhost:2019"}}
	err := client.Load(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", gotContentType)
	}

	if gotBody["admin"] == nil {
		t.Error("expected admin key in posted body")
	}
}

func TestClient_Load_Rejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid config: missing routes"))
	}))
	defer srv.Close()

	client := &Client{
		AdminAddr:  strings.TrimPrefix(srv.URL, "http://"),
		HTTPClient: srv.Client(),
	}

	err := client.Load(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	if !strings.Contains(err.Error(), "HTTP 400") {
		t.Errorf("expected error to contain 'HTTP 400', got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "invalid config: missing routes") {
		t.Errorf("expected error to contain response body, got: %s", err.Error())
	}
}

func TestClient_Load_ConnectionRefused(t *testing.T) {
	client := &Client{
		AdminAddr:  "localhost:0",
		HTTPClient: &http.Client{},
	}

	err := client.Load(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected connection error")
	}

	if !strings.Contains(err.Error(), "posting caddy config") {
		t.Errorf("expected wrapped connection error, got: %s", err.Error())
	}
}

func TestClient_Load_ContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &Client{
		AdminAddr:  strings.TrimPrefix(srv.URL, "http://"),
		HTTPClient: srv.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := client.Load(ctx, map[string]any{})
	if err == nil {
		t.Fatal("expected context canceled error")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context canceled error, got: %s", err.Error())
	}
}
