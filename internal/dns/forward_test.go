package dns

import (
	"testing"
)

const scutilFixture = `DNS configuration

resolver #1
  search domain[0] : home
  nameserver[0] : 192.168.1.1
  nameserver[1] : 192.168.1.2
  if_index : 6 (en0)
  flags    : Request A records
  reach    : 0x00020002 (Reachable,Directly Reachable Address)

resolver #2
  domain   : local
  options  : mdns
  timeout  : 5
  flags    : Request A records
  reach    : 0x00000000 (Not Reachable)
  order    : 300000

resolver #3
  domain   : 254.169.in-addr.arpa
  options  : mdns
  timeout  : 5
  flags    : Request A records
  reach    : 0x00000000 (Not Reachable)
  order    : 300200

DNS configuration (for scoped queries)

resolver #1
  search domain[0] : home
  nameserver[0] : 192.168.1.1
  nameserver[1] : 8.8.8.8
  if_index : 6 (en0)
  flags    : Scoped, Request A records
  reach    : 0x00020002 (Reachable,Directly Reachable Address)
`

func TestParseScutilDNS(t *testing.T) {
	servers := parseScutilDNS(scutilFixture)

	// Should find 192.168.1.1, 192.168.1.2, and 8.8.8.8 (deduplicated).
	expected := []string{"192.168.1.1:53", "192.168.1.2:53", "8.8.8.8:53"}

	if len(servers) != len(expected) {
		t.Fatalf("expected %d servers, got %d: %v", len(expected), len(servers), servers)
	}

	for i, want := range expected {
		if servers[i] != want {
			t.Errorf("servers[%d] = %q, want %q", i, servers[i], want)
		}
	}
}

func TestParseScutilDNS_FiltersLoopback(t *testing.T) {
	input := `DNS configuration

resolver #1
  nameserver[0] : 127.0.0.1
  nameserver[1] : 10.0.0.1
`

	servers := parseScutilDNS(input)

	if len(servers) != 1 {
		t.Fatalf("expected 1 server (loopback filtered), got %d: %v", len(servers), servers)
	}
	if servers[0] != "10.0.0.1:53" {
		t.Errorf("expected 10.0.0.1:53, got %q", servers[0])
	}
}

func TestParseScutilDNS_Empty(t *testing.T) {
	servers := parseScutilDNS("")

	if len(servers) != 0 {
		t.Errorf("expected empty result, got %v", servers)
	}
}
