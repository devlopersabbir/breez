package dns

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// Config holds DNS server configuration.
type Config struct {
	Domain   string // e.g. "breez.local"
	BindAddr string // e.g. "127.0.0.1:53" or "127.0.0.1:5354"
	TargetIP net.IP // default: 127.0.0.1
	Upstream string // e.g. "1.1.1.1:53" or "8.8.8.8:53"
}

// Server is a local DNS resolver for *.domain queries.
type Server struct {
	cfg        Config
	udpServer  *dns.Server
	tcpServer  *dns.Server
	upstreamCl *dns.Client
	mu         sync.Mutex
	running    bool
}

// NewServer creates a new DNS server instance.
func NewServer(cfg Config) *Server {
	if cfg.Domain == "" {
		cfg.Domain = "breez.local"
	}
	if cfg.BindAddr == "" {
		cfg.BindAddr = "127.0.0.1:5354"
	}
	if cfg.TargetIP == nil {
		cfg.TargetIP = net.ParseIP("127.0.0.1")
	}
	if cfg.Upstream == "" {
		cfg.Upstream = "1.1.1.1:53"
	}

	return &Server{
		cfg: cfg,
		upstreamCl: &dns.Client{
			Timeout: 2 * time.Second,
		},
	}
}

// Start runs both UDP and TCP DNS listeners. It blocks until ctx is done or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	mux := dns.NewServeMux()
	mux.HandleFunc(".", s.handleQuery)

	s.udpServer = &dns.Server{
		Addr:    s.cfg.BindAddr,
		Net:     "udp",
		Handler: mux,
	}

	s.tcpServer = &dns.Server{
		Addr:    s.cfg.BindAddr,
		Net:     "tcp",
		Handler: mux,
	}

	errCh := make(chan error, 2)

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	go func() {
		if err := s.udpServer.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("DNS UDP listener error: %w", err)
		}
	}()

	go func() {
		if err := s.tcpServer.ListenAndServe(); err != nil {
			errCh <- fmt.Errorf("DNS TCP listener error: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		s.Stop()
		return nil
	case err := <-errCh:
		s.Stop()
		return err
	}
}

// Stop shuts down the DNS server gracefully.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	if s.udpServer != nil {
		_ = s.udpServer.Shutdown()
	}
	if s.tcpServer != nil {
		_ = s.tcpServer.Shutdown()
	}
}

func (s *Server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	msg := new(dns.Msg)
	msg.SetReply(r)
	msg.Authoritative = true
	msg.RecursionAvailable = true

	suffix := strings.ToLower(s.cfg.Domain)
	if !strings.HasSuffix(suffix, ".") {
		suffix = suffix + "."
	}

	for _, q := range r.Question {
		qName := strings.ToLower(q.Name)

		// Check if query matches *.domain or domain
		if strings.HasSuffix(qName, suffix) || qName == suffix {
			switch q.Qtype {
			case dns.TypeA:
				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					A: s.cfg.TargetIP.To4(),
				}
				msg.Answer = append(msg.Answer, rr)

			case dns.TypeAAAA:
				rr := &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					AAAA: net.IPv6loopback,
				}
				msg.Answer = append(msg.Answer, rr)

			case dns.TypeTXT:
				rr := &dns.TXT{
					Hdr: dns.RR_Header{
						Name:   q.Name,
						Rrtype: dns.TypeTXT,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					Txt: []string{"breez local dns resolver"},
				}
				msg.Answer = append(msg.Answer, rr)
			}
		} else if s.cfg.Upstream != "" {
			// Forward non-local queries to upstream resolver
			upstreamResp, _, err := s.upstreamCl.Exchange(r, s.cfg.Upstream)
			if err == nil && upstreamResp != nil {
				_ = w.WriteMsg(upstreamResp)
				return
			}
		}
	}

	if len(msg.Answer) == 0 && msg.Rcode == dns.RcodeSuccess {
		// If query wasn't matched and upstream didn't answer, return NXDOMAIN
		if len(r.Question) > 0 {
			qName := strings.ToLower(r.Question[0].Name)
			if !strings.HasSuffix(qName, suffix) && qName != suffix {
				msg.Rcode = dns.RcodeNameError
			}
		}
	}

	if err := w.WriteMsg(msg); err != nil {
		log.Printf("[DNS] Failed to write response: %v", err)
	}
}
