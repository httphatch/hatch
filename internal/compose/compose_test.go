package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractHostPort(t *testing.T) {
	tests := []struct {
		name string
		raw  any
		want int
		ok   bool
	}{
		{"container only", "3000", 3000, true},
		{"host:container", "8080:80", 8080, true},
		{"ip:host:container", "127.0.0.1:3000:3000", 3000, true},
		{"with protocol", "8080:80/tcp", 8080, true},
		{"range", "3000-3005:3000-3005", 0, false},
		{"int value", 5432, 5432, true},
		{"empty string", "", 0, false},
		{"map published int", map[string]any{"target": 80, "published": 8080}, 8080, true},
		{"map published string", map[string]any{"target": "80", "published": "9090"}, 9090, true},
		{"map no published", map[string]any{"target": 80}, 0, false},
		{"unsupported type", 3.14, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractHostPort(tt.raw)
			if ok != tt.ok {
				t.Errorf("extractHostPort(%v) ok = %v, want %v", tt.raw, ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("extractHostPort(%v) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseComposeFile(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    []DiscoveredService
		wantErr bool
	}{
		{
			name: "single service with port",
			yaml: `services:
  web:
    image: nginx
    ports:
      - "8080:80"
`,
			want: []DiscoveredService{
				{Name: "web", Port: 8080},
			},
		},
		{
			name: "multiple services",
			yaml: `services:
  api:
    image: node
    ports:
      - "3000"
  db:
    image: postgres
    ports:
      - "5432:5432"
`,
			want: []DiscoveredService{
				{Name: "api", Port: 3000},
				{Name: "db", Port: 5432},
			},
		},
		{
			name: "service without ports skipped",
			yaml: `services:
  web:
    image: nginx
    ports:
      - "8080:80"
  worker:
    image: node
`,
			want: []DiscoveredService{
				{Name: "web", Port: 8080},
			},
		},
		{
			name: "long syntax ports",
			yaml: `services:
  web:
    image: nginx
    ports:
      - target: 80
        published: 8080
`,
			want: []DiscoveredService{
				{Name: "web", Port: 8080},
			},
		},
		{
			name: "uses first valid port",
			yaml: `services:
  web:
    image: nginx
    ports:
      - "3000-3005:3000-3005"
      - "8080:80"
`,
			want: []DiscoveredService{
				{Name: "web", Port: 8080},
			},
		},
		{
			name:    "invalid yaml",
			yaml:    `{{{`,
			wantErr: true,
		},
		{
			name: "no services with ports",
			yaml: `services:
  worker:
    image: node
`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "docker-compose.yml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0644); err != nil {
				t.Fatalf("writing test file: %v", err)
			}

			got, err := parseComposeFile(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseComposeFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %d services, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("service[%d].Name = %q, want %q", i, got[i].Name, tt.want[i].Name)
				}
				if got[i].Port != tt.want[i].Port {
					t.Errorf("service[%d].Port = %d, want %d", i, got[i].Port, tt.want[i].Port)
				}
			}
		})
	}
}

func TestDiscover(t *testing.T) {
	tests := []struct {
		name  string
		setup func(dir string)
		want  []DiscoveredService
	}{
		{
			name: "root compose file",
			setup: func(dir string) {
				writeFile(t, filepath.Join(dir, "docker-compose.yml"), `services:
  web:
    ports:
      - "3000"
`)
			},
			want: []DiscoveredService{
				{Name: "web", Port: 3000, Source: "docker-compose.yml"},
			},
		},
		{
			name: "child directory compose file",
			setup: func(dir string) {
				mkdirAll(t, filepath.Join(dir, "frontend"), 0755)
				writeFile(t, filepath.Join(dir, "frontend", "docker-compose.yml"), `services:
  web:
    ports:
      - "3000"
`)
			},
			want: []DiscoveredService{
				{Name: "frontend", Port: 3000, Source: "frontend/docker-compose.yml"},
			},
		},
		{
			name: "both root and child",
			setup: func(dir string) {
				writeFile(t, filepath.Join(dir, "compose.yml"), `services:
  api:
    ports:
      - "4000"
`)
				mkdirAll(t, filepath.Join(dir, "frontend"), 0755)
				writeFile(t, filepath.Join(dir, "frontend", "compose.yaml"), `services:
  web:
    ports:
      - "3000"
`)
			},
			want: []DiscoveredService{
				{Name: "api", Port: 4000, Source: "compose.yml"},
				{Name: "frontend", Port: 3000, Source: "frontend/compose.yaml"},
			},
		},
		{
			name: "nested too deep ignored",
			setup: func(dir string) {
				deep := filepath.Join(dir, "a", "b")
				mkdirAll(t,deep, 0755)
				writeFile(t, filepath.Join(deep, "docker-compose.yml"), `services:
  web:
    ports:
      - "3000"
`)
			},
			want: nil,
		},
		{
			name:  "no compose files",
			setup: func(dir string) {},
			want:  nil,
		},
		{
			name: "hidden dirs skipped",
			setup: func(dir string) {
				mkdirAll(t, filepath.Join(dir, ".hidden"), 0755)
				writeFile(t, filepath.Join(dir, ".hidden", "docker-compose.yml"), `services:
  web:
    ports:
      - "3000"
`)
			},
			want: nil,
		},
		{
			name: "compose.yaml preferred after docker-compose.yml",
			setup: func(dir string) {
				writeFile(t, filepath.Join(dir, "docker-compose.yml"), `services:
  first:
    ports:
      - "3000"
`)
				writeFile(t, filepath.Join(dir, "compose.yml"), `services:
  second:
    ports:
      - "4000"
`)
			},
			want: []DiscoveredService{
				{Name: "first", Port: 3000, Source: "docker-compose.yml"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			got, err := Discover(dir)
			if err != nil {
				t.Fatalf("Discover() error = %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("got %d services, want %d: %+v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i].Name != tt.want[i].Name {
					t.Errorf("service[%d].Name = %q, want %q", i, got[i].Name, tt.want[i].Name)
				}
				if got[i].Port != tt.want[i].Port {
					t.Errorf("service[%d].Port = %d, want %d", i, got[i].Port, tt.want[i].Port)
				}
				if got[i].Source != tt.want[i].Source {
					t.Errorf("service[%d].Source = %q, want %q", i, got[i].Source, tt.want[i].Source)
				}
			}
		})
	}
}

func mkdirAll(t *testing.T, path string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(path, perm); err != nil {
		t.Fatalf("creating %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
