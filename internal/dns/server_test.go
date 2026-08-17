package dns

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestDNSServerResolution(t *testing.T) {
	// Pick an available high port for testing
	cfg := Config{
		Domain:   "breez.local",
		BindAddr: "127.0.0.1:15354",
		TargetIP: net.ParseIP("127.0.0.1"),
		Upstream: "1.1.1.1:53",
	}

	server := NewServer(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	// Give server time to bind
	time.Sleep(100 * time.Millisecond)

	c := &dns.Client{Timeout: 2 * time.Second}

	// 1. Test A Record Query for myapp.breez.local
	m := new(dns.Msg)
	m.SetQuestion("myapp.breez.local.", dns.TypeA)

	in, _, err := c.Exchange(m, cfg.BindAddr)
	if err != nil {
		t.Fatalf("DNS query failed: %v", err)
	}

	if len(in.Answer) == 0 {
		t.Fatalf("expected at least 1 answer record, got 0")
	}

	aRecord, ok := in.Answer[0].(*dns.A)
	if !ok {
		t.Fatalf("expected *dns.A record, got %T", in.Answer[0])
	}

	if aRecord.A.String() != "127.0.0.1" {
		t.Fatalf("expected IP 127.0.0.1, got %s", aRecord.A.String())
	}

	// 2. Test AAAA Record Query for api.sub.breez.local
	m = new(dns.Msg)
	m.SetQuestion("api.sub.breez.local.", dns.TypeAAAA)

	in, _, err = c.Exchange(m, cfg.BindAddr)
	if err != nil {
		t.Fatalf("DNS AAAA query failed: %v", err)
	}

	if len(in.Answer) == 0 {
		t.Fatalf("expected AAAA record, got 0")
	}

	aaaaRecord, ok := in.Answer[0].(*dns.AAAA)
	if !ok {
		t.Fatalf("expected *dns.AAAA record, got %T", in.Answer[0])
	}

	if aaaaRecord.AAAA.String() != "::1" {
		t.Fatalf("expected ::1 IPv6 loopback, got %s", aaaaRecord.AAAA.String())
	}

	cancel()
	server.Stop()
}
