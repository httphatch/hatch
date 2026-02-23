package tunnel

import (
	"fmt"
	"testing"

	"github.com/httphatch/hatch/internal/config"
)

type mockResolver struct {
	hostnames map[string][]string
	err       error
}

func (m *mockResolver) GetTunnelHostnames(apiToken, accountID, tunnelName string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.hostnames[tunnelName], nil
}

func TestResolveTunnelDomains(t *testing.T) {
	t.Run("resolves named tunnel with token", func(t *testing.T) {
		cfg := config.Config{
			Settings: config.Settings{
				CloudflareToken: "test-token",
			},
			Projects: map[string]config.Project{
				"myapp": {
					Enabled: true,
					Services: map[string]config.Service{
						"web": {
							Proxy:  "http://localhost:3000",
							Tunnel: "my-tunnel",
						},
					},
				},
			},
		}

		resolver := &mockResolver{
			hostnames: map[string][]string{
				"my-tunnel": {"myapp.example.com", "api.example.com"},
			},
		}

		result := ResolveTunnelDomains(cfg, resolver)

		domains, ok := result["myapp/web"]
		if !ok {
			t.Fatal("expected myapp/web in result")
		}
		if len(domains) != 2 {
			t.Fatalf("expected 2 domains, got %d", len(domains))
		}
		if domains[0] != "myapp.example.com" {
			t.Errorf("expected myapp.example.com, got %s", domains[0])
		}
		if domains[1] != "api.example.com" {
			t.Errorf("expected api.example.com, got %s", domains[1])
		}
	})

	t.Run("skips quick tunnels", func(t *testing.T) {
		cfg := config.Config{
			Settings: config.Settings{
				CloudflareToken: "test-token",
			},
			Projects: map[string]config.Project{
				"myapp": {
					Enabled: true,
					Services: map[string]config.Service{
						"web": {
							Proxy:  "http://localhost:3000",
							Tunnel: "true",
						},
					},
				},
			},
		}

		resolver := &mockResolver{
			hostnames: map[string][]string{},
		}

		result := ResolveTunnelDomains(cfg, resolver)
		if len(result) != 0 {
			t.Errorf("expected empty result for quick tunnel, got %v", result)
		}
	})

	t.Run("skips named tunnels without token", func(t *testing.T) {
		cfg := config.Config{
			Projects: map[string]config.Project{
				"myapp": {
					Enabled: true,
					Services: map[string]config.Service{
						"web": {
							Proxy:  "http://localhost:3000",
							Tunnel: "my-tunnel",
						},
					},
				},
			},
		}

		resolver := &mockResolver{
			hostnames: map[string][]string{
				"my-tunnel": {"myapp.example.com"},
			},
		}

		result := ResolveTunnelDomains(cfg, resolver)
		if len(result) != 0 {
			t.Errorf("expected empty result without token, got %v", result)
		}
	})

	t.Run("skips disabled projects", func(t *testing.T) {
		cfg := config.Config{
			Settings: config.Settings{
				CloudflareToken: "test-token",
			},
			Projects: map[string]config.Project{
				"myapp": {
					Enabled: false,
					Services: map[string]config.Service{
						"web": {
							Proxy:  "http://localhost:3000",
							Tunnel: "my-tunnel",
						},
					},
				},
			},
		}

		resolver := &mockResolver{
			hostnames: map[string][]string{
				"my-tunnel": {"myapp.example.com"},
			},
		}

		result := ResolveTunnelDomains(cfg, resolver)
		if len(result) != 0 {
			t.Errorf("expected empty result for disabled project, got %v", result)
		}
	})

	t.Run("uses project-level token over global", func(t *testing.T) {
		cfg := config.Config{
			Settings: config.Settings{
				CloudflareToken: "global-token",
			},
			Projects: map[string]config.Project{
				"myapp": {
					Enabled:         true,
					CloudflareToken: "project-token",
					Services: map[string]config.Service{
						"web": {
							Proxy:  "http://localhost:3000",
							Tunnel: "my-tunnel",
						},
					},
				},
			},
		}

		var calledWithToken string
		resolver := &mockResolver{
			hostnames: map[string][]string{
				"my-tunnel": {"myapp.example.com"},
			},
		}
		// Override with a custom resolver to capture the token.
		captureResolver := &tokenCapture{
			inner:    resolver,
			captured: &calledWithToken,
		}

		result := ResolveTunnelDomains(cfg, captureResolver)
		if len(result) != 1 {
			t.Fatalf("expected 1 result, got %d", len(result))
		}
		if calledWithToken != "project-token" {
			t.Errorf("expected project-token, got %s", calledWithToken)
		}
	})

	t.Run("API error returns partial results", func(t *testing.T) {
		cfg := config.Config{
			Settings: config.Settings{
				CloudflareToken: "test-token",
			},
			Projects: map[string]config.Project{
				"good": {
					Enabled: true,
					Services: map[string]config.Service{
						"web": {
							Proxy:  "http://localhost:3000",
							Tunnel: "good-tunnel",
						},
					},
				},
				"bad": {
					Enabled: true,
					Services: map[string]config.Service{
						"web": {
							Proxy:  "http://localhost:4000",
							Tunnel: "bad-tunnel",
						},
					},
				},
			},
		}

		resolver := &selectiveResolver{
			hostnames: map[string][]string{
				"good-tunnel": {"good.example.com"},
			},
			failOn: "bad-tunnel",
		}

		result := ResolveTunnelDomains(cfg, resolver)
		if _, ok := result["good/web"]; !ok {
			t.Error("expected good/web in partial result")
		}
		if _, ok := result["bad/web"]; ok {
			t.Error("expected bad/web to be absent from partial result")
		}
	})
}

type tokenCapture struct {
	inner    HostnameResolver
	captured *string
}

func (tc *tokenCapture) GetTunnelHostnames(apiToken, accountID, tunnelName string) ([]string, error) {
	*tc.captured = apiToken
	return tc.inner.GetTunnelHostnames(apiToken, accountID, tunnelName)
}

type selectiveResolver struct {
	hostnames map[string][]string
	failOn    string
}

func (s *selectiveResolver) GetTunnelHostnames(apiToken, accountID, tunnelName string) ([]string, error) {
	if tunnelName == s.failOn {
		return nil, fmt.Errorf("API error for tunnel %s", tunnelName)
	}
	return s.hostnames[tunnelName], nil
}
