package session

import "testing"

func TestSubstituteCommand(t *testing.T) {
	ports := map[string]int{
		"web": 51234,
		"api": 51235,
	}

	tests := []struct {
		name    string
		cmd     string
		service string
		want    string
	}{
		{
			name:    "simple port",
			cmd:     "npm run dev -- --port {{port}}",
			service: "web",
			want:    "npm run dev -- --port 51234",
		},
		{
			name:    "cross-service reference",
			cmd:     "npm run dev -- --api-port {{port:api}}",
			service: "web",
			want:    "npm run dev -- --api-port 51235",
		},
		{
			name:    "both self and cross-service",
			cmd:     "--port {{port}} --api {{port:api}}",
			service: "web",
			want:    "--port 51234 --api 51235",
		},
		{
			name:    "no template",
			cmd:     "npm run dev",
			service: "web",
			want:    "npm run dev",
		},
		{
			name:    "unknown cross-service left unresolved",
			cmd:     "--port {{port:unknown}}",
			service: "web",
			want:    "--port {{port:unknown}}",
		},
		{
			name:    "unknown self service left unresolved",
			cmd:     "--port {{port}}",
			service: "missing",
			want:    "--port {{port}}",
		},
		{
			name:    "multiple occurrences",
			cmd:     "{{port}} and {{port}}",
			service: "web",
			want:    "51234 and 51234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubstituteCommand(tt.cmd, tt.service, ports)
			if got != tt.want {
				t.Errorf("SubstituteCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubstituteEnv(t *testing.T) {
	ports := map[string]int{"web": 8080}
	env := []string{
		"PATH=/usr/bin",
		"PORT={{port}}",
		"API_PORT={{port:web}}",
		"NOVALUE",
	}

	got := SubstituteEnv(env, "web", ports)

	want := []string{
		"PATH=/usr/bin",
		"PORT=8080",
		"API_PORT=8080",
		"NOVALUE",
	}

	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("env[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestHasPortTemplate(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"npm run dev -- --port {{port}}", true},
		{"{{port:api}}", true},
		{"npm run dev", false},
		{"{port}", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := HasPortTemplate(tt.input); got != tt.want {
				t.Errorf("HasPortTemplate(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
