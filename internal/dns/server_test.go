package dns

import (
	"net"
	"strconv"
	"testing"

	mdns "github.com/miekg/dns"
)

// startTestServer creates and starts a DNS server on a random port
// for testing. It returns the server and the address it's listening on.
func startTestServer(t *testing.T, tld string, upstreams []string) (*Server, string) {
	t.Helper()

	// Find a free port.
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding free port: %v", err)
	}
	port := conn.LocalAddr().(*net.UDPAddr).Port
	_ = conn.Close()

	cfg := ServerConfig{
		TLD:      tld,
		ListenIP: "127.0.0.1",
		Port:     port,
	}

	srv := newServerWithUpstreams(cfg, upstreams)
	if err := srv.Start(); err != nil {
		t.Fatalf("starting server: %v", err)
	}

	t.Cleanup(func() {
		_ = srv.Stop()
	})

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	return srv, addr
}

func TestServer_StartStop(t *testing.T) {
	_, _ = startTestServer(t, "test", nil)
	// If we get here without error, start/stop works.
}

func TestServer_ResolvesMatchingTLD(t *testing.T) {
	_, addr := startTestServer(t, "test", nil)

	client := &mdns.Client{Net: "udp"}
	msg := new(mdns.Msg)
	msg.SetQuestion("myapp.test.", mdns.TypeA)

	resp, _, err := client.Exchange(msg, addr)
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(resp.Answer))
	}

	a, ok := resp.Answer[0].(*mdns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", resp.Answer[0])
	}

	if !a.A.Equal(net.ParseIP("127.0.0.1")) {
		t.Errorf("expected 127.0.0.1, got %s", a.A)
	}
}

func TestServer_ResolvesWildcard(t *testing.T) {
	_, addr := startTestServer(t, "test", nil)

	client := &mdns.Client{Net: "udp"}
	msg := new(mdns.Msg)
	msg.SetQuestion("a.b.test.", mdns.TypeA)

	resp, _, err := client.Exchange(msg, addr)
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(resp.Answer))
	}

	a, ok := resp.Answer[0].(*mdns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", resp.Answer[0])
	}

	if !a.A.Equal(net.ParseIP("127.0.0.1")) {
		t.Errorf("expected 127.0.0.1, got %s", a.A)
	}
}

func TestServer_AAAARecord(t *testing.T) {
	_, addr := startTestServer(t, "test", nil)

	client := &mdns.Client{Net: "udp"}
	msg := new(mdns.Msg)
	msg.SetQuestion("myapp.test.", mdns.TypeAAAA)

	resp, _, err := client.Exchange(msg, addr)
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(resp.Answer))
	}

	aaaa, ok := resp.Answer[0].(*mdns.AAAA)
	if !ok {
		t.Fatalf("expected AAAA record, got %T", resp.Answer[0])
	}

	if !aaaa.AAAA.Equal(net.ParseIP("::1")) {
		t.Errorf("expected ::1, got %s", aaaa.AAAA)
	}
}

func TestServer_NonMatchingForwarded(t *testing.T) {
	// Start a mock "upstream" DNS server that responds to any query.
	upstreamConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("finding free port: %v", err)
	}
	upstreamPort := upstreamConn.LocalAddr().(*net.UDPAddr).Port
	_ = upstreamConn.Close()

	upstreamAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(upstreamPort))

	upstreamMux := mdns.NewServeMux()
	upstreamMux.HandleFunc(".", func(w mdns.ResponseWriter, r *mdns.Msg) {
		msg := new(mdns.Msg)
		msg.SetReply(r)
		msg.Answer = append(msg.Answer, &mdns.A{
			Hdr: mdns.RR_Header{
				Name:   r.Question[0].Name,
				Rrtype: mdns.TypeA,
				Class:  mdns.ClassINET,
				Ttl:    300,
			},
			A: net.ParseIP("93.184.216.34").To4(),
		})
		_ = w.WriteMsg(msg)
	})

	upstream := &mdns.Server{
		Addr:    upstreamAddr,
		Net:     "udp",
		Handler: upstreamMux,
	}

	upstreamReady := make(chan struct{})
	upstream.NotifyStartedFunc = func() { close(upstreamReady) }

	go func() { _ = upstream.ListenAndServe() }()
	<-upstreamReady
	defer func() { _ = upstream.Shutdown() }()

	// Start our DNS server with the mock upstream.
	_, addr := startTestServer(t, "test", []string{upstreamAddr})

	client := &mdns.Client{Net: "udp"}
	msg := new(mdns.Msg)
	msg.SetQuestion("example.com.", mdns.TypeA)

	resp, _, err := client.Exchange(msg, addr)
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if len(resp.Answer) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(resp.Answer))
	}

	a, ok := resp.Answer[0].(*mdns.A)
	if !ok {
		t.Fatalf("expected A record, got %T", resp.Answer[0])
	}

	if !a.A.Equal(net.ParseIP("93.184.216.34")) {
		t.Errorf("expected upstream response 93.184.216.34, got %s", a.A)
	}
}
