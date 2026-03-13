package cmd

import "testing"

func TestStartupLabel(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{`{"level":"info","time":"2026-01-01T00:00:00Z","message":"dns server started"}`, "Starting DNS server"},
		{`{"level":"info","time":"2026-01-01T00:00:00Z","message":"caddy server started"}`, "Starting Caddy server"},
		{`{"level":"info","domains":2,"services":2,"time":"2026-01-01T00:00:00Z","message":"resolved tunnel domains"}`, "Resolving tunnel domains"},
		{`{"level":"info","time":"2026-01-01T00:00:00Z","message":"caddy config loaded"}`, "Loading Caddy config"},
		{`{"level":"info","time":"2026-01-01T00:00:00Z","message":"health checker started"}`, "Starting health checker"},
		{`{"level":"info","time":"2026-01-01T00:00:00Z","message":"process manager started"}`, "Starting process manager"},
		{`{"level":"info","time":"2026-01-01T00:00:00Z","message":"tunnel manager started"}`, "Starting tunnel manager"},
		{`{"level":"info","addr":"127.0.0.1:42824","time":"2026-01-01T00:00:00Z","message":"api server started"}`, "Starting API server"},
		{`{"level":"info","time":"2026-01-01T00:00:00Z","message":"config watcher started"}`, "Starting config watcher"},
		{`{"level":"info","time":"2026-01-01T00:00:00Z","message":"daemon running"}`, "Ready"},
		// Lines that should not match.
		{`{"level":"info","time":"2026-01-01T00:00:00Z","message":"pid file written"}`, ""},
		{`{"level":"info","project":"myapp","service":"web","source":"stdout","time":"2026-01-01T00:00:00Z","message":"listening on :3000"}`, ""},
		{`not json at all`, ""},
		{``, ""},
	}

	for _, tt := range tests {
		got := startupLabel(tt.line)
		if got != tt.want {
			t.Errorf("startupLabel(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}
