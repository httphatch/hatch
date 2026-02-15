package dns

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	mdns "github.com/miekg/dns"
)

// Server is an embedded DNS server that resolves wildcard queries for
// the configured TLD to loopback and forwards all other queries
// upstream.
type Server struct {
	cfg       ServerConfig
	server    *mdns.Server
	upstreams []string
	tldSuffix string // e.g. ".test."

	mu      sync.Mutex
	started bool
}

// NewServer creates a DNS server for the given config. It discovers
// system DNS servers for forwarding non-matching queries.
func NewServer(cfg ServerConfig) (*Server, error) {
	upstreams, err := SystemDNSServers()
	if err != nil {
		// Discovery failed but returned fallback servers — log and continue.
		_ = err
	}

	tld := strings.TrimPrefix(cfg.TLD, ".")
	return &Server{
		cfg:       cfg,
		upstreams: upstreams,
		tldSuffix: "." + tld + ".",
	}, nil
}

// newServerWithUpstreams creates a Server with explicit upstreams,
// used in tests to avoid system DNS discovery.
func newServerWithUpstreams(cfg ServerConfig, upstreams []string) *Server {
	tld := strings.TrimPrefix(cfg.TLD, ".")
	return &Server{
		cfg:       cfg,
		upstreams: upstreams,
		tldSuffix: "." + tld + ".",
	}
}

// Start begins listening for DNS queries on the configured address.
// It blocks until the server is ready to accept connections, then
// returns. Use Stop to shut down.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.ListenIP, s.cfg.Port)

	mux := mdns.NewServeMux()
	mux.HandleFunc(".", s.handleDNS)

	s.server = &mdns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: mux,
		NotifyStartedFunc: func() {
			s.mu.Lock()
			s.started = true
			s.mu.Unlock()
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	// Wait for the server to start or fail.
	for i := 0; i < 100; i++ {
		s.mu.Lock()
		ready := s.started
		s.mu.Unlock()
		if ready {
			return nil
		}

		select {
		case err := <-errCh:
			return fmt.Errorf("dns server failed to start: %w", err)
		default:
		}

		time.Sleep(5 * time.Millisecond)
	}

	return fmt.Errorf("dns server did not start within timeout")
}

// Stop gracefully shuts down the DNS server.
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown()
}

// handleDNS processes incoming DNS queries. If the query name matches
// *.<tld>, it responds with A → 127.0.0.1 or AAAA → ::1. Otherwise
// it forwards the query to upstream DNS servers.
func (s *Server) handleDNS(w mdns.ResponseWriter, r *mdns.Msg) {
	if len(r.Question) == 0 {
		return
	}

	q := r.Question[0]
	name := strings.ToLower(q.Name)

	// Check if the query matches our TLD.
	if strings.HasSuffix(name, s.tldSuffix) {
		s.respondLocal(w, r, q)
		return
	}

	// Forward to upstream.
	s.forwardQuery(w, r)
}

// respondLocal writes a DNS response mapping the queried name to
// loopback addresses.
func (s *Server) respondLocal(w mdns.ResponseWriter, r *mdns.Msg, q mdns.Question) {
	msg := new(mdns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true

	switch q.Qtype {
	case mdns.TypeA:
		msg.Answer = append(msg.Answer, &mdns.A{
			Hdr: mdns.RR_Header{
				Name:   q.Name,
				Rrtype: mdns.TypeA,
				Class:  mdns.ClassINET,
				Ttl:    0,
			},
			A: net.ParseIP("127.0.0.1").To4(),
		})
	case mdns.TypeAAAA:
		msg.Answer = append(msg.Answer, &mdns.AAAA{
			Hdr: mdns.RR_Header{
				Name:   q.Name,
				Rrtype: mdns.TypeAAAA,
				Class:  mdns.ClassINET,
				Ttl:    0,
			},
			AAAA: net.ParseIP("::1"),
		})
	}

	_ = w.WriteMsg(msg)
}

// forwardQuery sends the DNS query to upstream servers and returns
// the first successful response.
func (s *Server) forwardQuery(w mdns.ResponseWriter, r *mdns.Msg) {
	client := &mdns.Client{
		Net:     "udp",
		Timeout: 5 * time.Second,
	}

	for _, upstream := range s.upstreams {
		resp, _, err := client.Exchange(r, upstream)
		if err != nil {
			continue
		}
		resp.Id = r.Id
		_ = w.WriteMsg(resp)
		return
	}

	// All upstreams failed — return SERVFAIL.
	msg := new(mdns.Msg)
	msg.SetReply(r)
	msg.Rcode = mdns.RcodeServerFailure
	_ = w.WriteMsg(msg)
}
