package daemon

import (
	"fmt"
	"net"
	"testing"
)

func TestParseLsofOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		port    int
		want    *PortInfo
	}{
		{
			name: "typical nginx",
			output: "COMMAND  PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"nginx   1234   root    6u  IPv4  12345      0t0  TCP *:443 (LISTEN)\n",
			port: 443,
			want: &PortInfo{Process: "nginx", PID: 1234},
		},
		{
			name:   "empty output",
			output: "",
			port:   443,
			want:   nil,
		},
		{
			name:   "header only",
			output: "COMMAND  PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n",
			port:   443,
			want:   nil,
		},
		{
			name: "httpd on port 80",
			output: "COMMAND  PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"httpd   5678   root    4u  IPv6  67890      0t0  TCP *:80 (LISTEN)\n",
			port: 80,
			want: &PortInfo{Process: "httpd", PID: 5678},
		},
		{
			name: "multiple lines takes first",
			output: "COMMAND  PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"caddy   1000   paul    5u  IPv4  11111      0t0  TCP *:443 (LISTEN)\n" +
				"caddy   1000   paul    6u  IPv6  22222      0t0  TCP *:443 (LISTEN)\n",
			port: 443,
			want: &PortInfo{Process: "caddy", PID: 1000},
		},
		{
			name: "outbound connections ignored",
			output: "COMMAND  PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"Slack   33068   paul   51u  IPv4  0x1234      0t0  TCP 192.168.10.5:51324->1.2.3.4:443 (ESTABLISHED)\n" +
				"Google  48801   paul   52u  IPv4  0x5678      0t0  TCP 192.168.10.5:51400->5.6.7.8:443 (ESTABLISHED)\n",
			port: 443,
			want: nil,
		},
		{
			name: "listen mixed with established",
			output: "COMMAND  PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"Slack   33068   paul   51u  IPv4  0x1234      0t0  TCP 192.168.10.5:51324->1.2.3.4:443 (ESTABLISHED)\n" +
				"nginx   1234    root    6u  IPv4  12345       0t0  TCP *:443 (LISTEN)\n",
			port: 443,
			want: &PortInfo{Process: "nginx", PID: 1234},
		},
		{
			name: "control characters in process name are sanitized",
			output: "COMMAND  PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"\x1b[31mevil\x1b[0m   999   root    6u  IPv4  12345      0t0  TCP *:80 (LISTEN)\n",
			port: 80,
			want: &PortInfo{Process: "?[31mevil?[0m", PID: 999},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLsofOutput(tt.output, tt.port)
			if tt.want == nil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("got nil, want %v", tt.want)
			}
			if got.Process != tt.want.Process {
				t.Errorf("Process: got %q, want %q", got.Process, tt.want.Process)
			}
			if got.PID != tt.want.PID {
				t.Errorf("PID: got %d, want %d", got.PID, tt.want.PID)
			}
		})
	}
}

func TestPortInfoString(t *testing.T) {
	tests := []struct {
		name string
		info PortInfo
		want string
	}{
		{
			name: "with PID",
			info: PortInfo{Process: "nginx", PID: 1234},
			want: "nginx (PID 1234)",
		},
		{
			name: "without PID",
			info: PortInfo{Process: "nginx", PID: 0},
			want: "nginx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckPort_Listening(t *testing.T) {
	// Listen on an ephemeral port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	port := ln.Addr().(*net.TCPAddr).Port

	info, err := CheckPort(port)
	if err != nil {
		t.Fatalf("CheckPort: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil PortInfo for listening port")
	}
	if info.PID == 0 {
		t.Error("expected non-zero PID")
	}
}

func TestCheckPort_Free(t *testing.T) {
	// Find a free port by briefly listening and then closing.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	info, err := CheckPort(port)
	if err != nil {
		t.Fatalf("CheckPort: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil PortInfo for free port, got %v", info)
	}
}

func TestPortInfoString_Format(t *testing.T) {
	info := &PortInfo{Process: "caddy", PID: 42}
	expected := "caddy (PID 42)"
	got := fmt.Sprintf("port :443 in use by %s", info)
	if got != "port :443 in use by "+expected {
		t.Errorf("got %q, want %q", got, "port :443 in use by "+expected)
	}
}
