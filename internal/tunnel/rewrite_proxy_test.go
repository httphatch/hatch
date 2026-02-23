package tunnel

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestRewriteProxy(t *testing.T) {
	// Simulate a Laravel+Vite setup: app server returns HTML with absolute
	// URLs pointing to a separate Vite dev server on a different port.
	var vitePort string
	viteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = fmt.Fprintf(w, `console.log("vite asset: %s")`, r.URL.Path)
	}))
	defer viteServer.Close()

	viteURL, _ := url.Parse(viteServer.URL)
	vitePort = viteURL.Port()

	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, `<!doctype html>
<html>
<head>
<script type="module" src="http://127.0.0.1:%s/@vite/client"></script>
<script type="module" src="http://127.0.0.1:%s/resources/js/app.tsx"></script>
</head>
<body><div id="root"></div></body>
</html>`, vitePort, vitePort)
			return
		}
		http.NotFound(w, r)
	}))
	defer appServer.Close()

	proxy, err := StartRewriteProxy(appServer.URL)
	if err != nil {
		t.Fatalf("starting proxy: %v", err)
	}
	defer proxy.Close()

	// First request: HTML page. URLs should be rewritten to relative paths
	// and the dev server should be discovered.
	resp, err := http.Get(proxy.Addr() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	got := string(body)

	if !strings.Contains(got, `src="/@vite/client"`) {
		t.Errorf("expected relative /@vite/client in HTML, got:\n%s", got)
	}
	if !strings.Contains(got, `src="/resources/js/app.tsx"`) {
		t.Errorf("expected relative /resources/js/app.tsx in HTML, got:\n%s", got)
	}
	if strings.Contains(got, "[::1]:") {
		t.Errorf("HTML still contains [::1] reference:\n%s", got)
	}

	// Second request: asset that 404s on app server. Should fall back to
	// the discovered Vite dev server.
	resp, err = http.Get(proxy.Addr() + "/@vite/client")
	if err != nil {
		t.Fatalf("GET /@vite/client: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for /@vite/client, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "vite asset: /@vite/client") {
		t.Errorf("expected vite asset content, got: %s", body)
	}

	// Third request: another asset path.
	resp, err = http.Get(proxy.Addr() + "/resources/js/app.tsx")
	if err != nil {
		t.Fatalf("GET /resources/js/app.tsx: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for /resources/js/app.tsx, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "vite asset: /resources/js/app.tsx") {
		t.Errorf("expected vite asset content, got: %s", body)
	}
}

func TestRewriteProxyWebSocketFallback(t *testing.T) {
	// Vite's HMR WebSocket connects to the page origin (tunnel URL).
	// The upstream (Laravel) doesn't handle WebSocket, so the proxy should
	// fall back to the discovered dev server (Vite).
	var vitePort string
	viteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a successful WebSocket upgrade check. A real test would
		// need a full WebSocket server, but we just verify the request
		// reaches the dev server by returning 200.
		if r.Header.Get("Upgrade") == "websocket" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ws-from-vite"))
			return
		}
		http.NotFound(w, r)
	}))
	defer viteServer.Close()

	viteURL, _ := url.Parse(viteServer.URL)
	vitePort = viteURL.Port()

	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && r.Header.Get("Upgrade") == "" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprintf(w, `<script src="http://127.0.0.1:%s/@vite/client"></script>`, vitePort)
			return
		}
		// Laravel returns 200 HTML for WebSocket requests (it doesn't know about WS).
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>not found</html>"))
	}))
	defer appServer.Close()

	proxy, err := StartRewriteProxy(appServer.URL)
	if err != nil {
		t.Fatalf("starting proxy: %v", err)
	}
	defer proxy.Close()

	// Load HTML to discover the dev server.
	resp, err := http.Get(proxy.Addr() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	_ = resp.Body.Close()

	// Simulate a WebSocket upgrade request. The upstream returns 200 (not 101),
	// so the fallback should retry against the Vite dev server.
	req, _ := http.NewRequest("GET", proxy.Addr()+"/?token=abc", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("WebSocket request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if !strings.Contains(string(body), "ws-from-vite") {
		t.Errorf("WebSocket request did not reach dev server, got status %d body: %s",
			resp.StatusCode, body)
	}
}

func TestRewriteProxyNonHTML(t *testing.T) {
	var port string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = fmt.Fprintf(w, `const url = "http://localhost:%s/api"`, port)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	port = u.Port()

	proxy, err := StartRewriteProxy(upstream.URL)
	if err != nil {
		t.Fatalf("starting proxy: %v", err)
	}
	defer proxy.Close()

	resp, err := http.Get(proxy.Addr() + "/test.js")
	if err != nil {
		t.Fatalf("GET proxy: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	got := string(body)

	want := fmt.Sprintf("http://localhost:%s/api", port)
	if !strings.Contains(got, want) {
		t.Errorf("non-HTML response was rewritten, got:\n%s", got)
	}
}

func TestRewriteProxySamePort(t *testing.T) {
	// When all localhost URLs use the same port as the upstream (e.g. plain
	// Vite without Laravel), no dev server fallback is needed. Assets are
	// served directly by the upstream.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			u, _ := url.Parse(r.Host)
			_ = u
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprintf(w, `<script src="http://localhost:%s/@vite/client"></script>`,
				strings.Split(r.Host, ":")[1])
			return
		}
		if r.URL.Path == "/@vite/client" {
			w.Header().Set("Content-Type", "application/javascript")
			_, _ = w.Write([]byte(`console.log("vite")`))
			return
		}
		http.NotFound(w, r)
	}))
	defer upstream.Close()

	proxy, err := StartRewriteProxy(upstream.URL)
	if err != nil {
		t.Fatalf("starting proxy: %v", err)
	}
	defer proxy.Close()

	// Load HTML to trigger rewriting.
	resp, err := http.Get(proxy.Addr() + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	_ = resp.Body.Close()

	// Asset request should succeed directly from upstream (no fallback needed).
	resp, err = http.Get(proxy.Addr() + "/@vite/client")
	if err != nil {
		t.Fatalf("GET /@vite/client: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "vite") {
		t.Errorf("expected vite content, got: %s", body)
	}
}

func TestLocalhostRegex(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: `<script src="http://localhost:3000/@vite/client"></script>`,
			want:  `<script src="/@vite/client"></script>`,
		},
		{
			input: `<script src="http://[::1]:5173/resources/js/app.tsx"></script>`,
			want:  `<script src="/resources/js/app.tsx"></script>`,
		},
		{
			input: `<script src="http://127.0.0.1:8080/main.js"></script>`,
			want:  `<script src="/main.js"></script>`,
		},
		{
			input: `new WebSocket("ws://localhost:3000/ws")`,
			want:  `new WebSocket("/ws")`,
		},
		{
			input: `<script src="https://cdn.example.com/lib.js"></script>`,
			want:  `<script src="https://cdn.example.com/lib.js"></script>`,
		},
		{
			input: `<script src="/already-relative.js"></script>`,
			want:  `<script src="/already-relative.js"></script>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := string(localhostRe.ReplaceAll([]byte(tt.input), nil))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
