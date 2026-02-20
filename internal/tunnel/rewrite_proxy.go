package tunnel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// maxRewriteBody is the maximum HTML response body size the rewrite proxy
// will buffer. Responses larger than this pass through without rewriting.
const maxRewriteBody = 10 << 20 // 10 MB

// localhostRe matches absolute http/ws URLs pointing to any localhost
// address on any port. Used to strip these prefixes from HTML responses
// so paths become relative (e.g. "/@vite/client").
var localhostRe = regexp.MustCompile(
	`(?:https?|wss?)://(?:localhost|127\.0\.0\.1|\[::1\]):\d+`,
)

// rewriteProxy sits between cloudflared and the upstream app server.
// It does two things:
//  1. Rewrites absolute localhost URLs in HTML responses to relative paths,
//     preventing PNA (Private Network Access) browser blocks.
//  2. Falls back to a discovered dev server (e.g. Vite) for asset requests
//     that 404 on the upstream. The dev server address is extracted from
//     localhost URLs found in the first HTML response.
type rewriteProxy struct {
	server   *http.Server
	listener net.Listener
	addr     string
}

// startRewriteProxy starts a local HTTP proxy that forwards requests to
// upstream, rewrites HTML responses, and falls back to a discovered dev
// server for 404'd assets. Returns the proxy's listen address for cloudflared.
func startRewriteProxy(upstream string) (*rewriteProxy, error) {
	target, err := url.Parse(upstream)
	if err != nil {
		return nil, fmt.Errorf("parsing upstream URL: %w", err)
	}

	ft := &devFallbackTransport{
		inner:    http.DefaultTransport,
		upstream: target,
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = ft
	proxy.ModifyResponse = func(resp *http.Response) error {
		ct := resp.Header.Get("Content-Type")
		if !strings.Contains(ct, "text/html") {
			return nil
		}
		if resp.Header.Get("Content-Encoding") != "" {
			return nil
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, maxRewriteBody))
		_ = resp.Body.Close()
		if err != nil {
			return fmt.Errorf("reading response body: %w", err)
		}

		// Extract dev server address from localhost URLs before rewriting.
		// In setups like Laravel+Vite, the HTML from the app server contains
		// script tags pointing to the Vite dev server on a different port.
		ft.discoverDevServer(body, target)

		body = localhostRe.ReplaceAll(body, nil)

		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
		return nil
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("creating listener: %w", err)
	}

	srv := &http.Server{Handler: proxy}
	go func() { _ = srv.Serve(listener) }()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return nil, fmt.Errorf("listener address is not TCP")
	}

	addr := fmt.Sprintf("http://127.0.0.1:%d", tcpAddr.Port)
	return &rewriteProxy{
		server:   srv,
		listener: listener,
		addr:     addr,
	}, nil
}

func (p *rewriteProxy) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = p.server.Shutdown(ctx)
	_ = p.listener.Close()
}

// devFallbackTransport wraps an http.RoundTripper and retries GET requests
// against a discovered dev server when the upstream returns 404. This handles
// setups where the app server (e.g. Laravel) serves HTML but a separate dev
// server (e.g. Vite on a different port) serves JS/CSS assets.
type devFallbackTransport struct {
	inner    http.RoundTripper
	upstream *url.URL
	devAddr  atomic.Value // stores *url.URL or nil
}

// discoverDevServer scans HTML bytes for localhost URLs and records the first
// one whose port differs from the upstream. This is the dev server (e.g. Vite).
func (t *devFallbackTransport) discoverDevServer(html []byte, upstream *url.URL) {
	if t.devAddr.Load() != nil {
		return // already discovered
	}

	matches := localhostRe.FindAll(html, -1)
	for _, match := range matches {
		u, err := url.Parse(string(match))
		if err != nil {
			continue
		}
		if u.Port() != upstream.Port() {
			t.devAddr.Store(u)
			return
		}
	}
}

func (t *devFallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Only retry GETs. POST/PUT/etc should not be retried.
	if req.Method != http.MethodGet {
		return resp, nil
	}

	// Decide whether to retry against the dev server:
	// - 404: asset not found on upstream (e.g. Vite files served by Laravel)
	// - WebSocket upgrade that didn't get 101: the upstream (e.g. Laravel)
	//   doesn't handle WebSocket, but the dev server (Vite HMR) does.
	shouldRetry := resp.StatusCode == http.StatusNotFound
	if !shouldRetry && isWebSocketUpgrade(req) && resp.StatusCode != http.StatusSwitchingProtocols {
		shouldRetry = true
	}

	if !shouldRetry {
		return resp, nil
	}

	dev, _ := t.devAddr.Load().(*url.URL)
	if dev == nil {
		return resp, nil
	}

	// The upstream can't serve this request and we know about a dev server.
	// Close the response and retry against the dev server.
	_ = resp.Body.Close()

	retry := req.Clone(req.Context())
	retry.URL.Scheme = dev.Scheme
	retry.URL.Host = dev.Host
	retry.Host = dev.Host
	return t.inner.RoundTrip(retry)
}

func isWebSocketUpgrade(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("Connection"), "upgrade") &&
		strings.EqualFold(req.Header.Get("Upgrade"), "websocket")
}
