package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// Server is an embedded DNS server that resolves *.orb.local domains
type Server struct {
	addr         string
	upstream     string
	projectName  string
	domain       string
	workDir      string // Working directory for hash generation
	useHashing   bool   // Enable directory-based hashing for DNS names
	server       *dns.Server
	docker       DockerClient
	cache        *cache
	mu           sync.RWMutex
	running      bool
	logger       Logger
}

// DockerClient interface for Docker operations
type DockerClient interface {
	GetContainerIP(ctx context.Context, projectName, containerName string) (string, error)
	GetContainerIPByHash(ctx context.Context, serviceName, hash string) (string, error)
	ListProjectContainers(ctx context.Context, projectName string) (map[string]string, error)
}

// Logger interface for logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// Config holds DNS server configuration
type Config struct {
	Addr        string        // Address to listen on (e.g., "127.0.0.1:5353")
	Upstream    string        // Upstream DNS server (e.g., "8.8.8.8:53")
	ProjectName string        // Docker compose project name
	Domain      string        // Domain to handle (e.g., "orb.local")
	WorkDir     string        // Working directory for hash generation
	UseHashing  bool          // Enable directory-based hashing (default: true)
	CacheTTL    time.Duration // Cache TTL (default: 30s)
	Docker      DockerClient  // Docker client
	Logger      Logger        // Logger
}

// NewServer creates a new DNS server
func NewServer(cfg Config) (*Server, error) {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:5353"
	}
	if cfg.Upstream == "" {
		cfg.Upstream = "8.8.8.8:53"
	}
	if cfg.Domain == "" {
		cfg.Domain = "orb.local"
	}
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 30 * time.Second
	}
	// Default to enabling hashing unless explicitly disabled
	useHashing := cfg.UseHashing
	if cfg.WorkDir != "" && !cfg.UseHashing {
		// Only disable if explicitly set to false with a workdir
		useHashing = false
	} else if cfg.WorkDir != "" {
		// Enable hashing by default when workdir is provided
		useHashing = true
	}

	s := &Server{
		addr:        cfg.Addr,
		upstream:    cfg.Upstream,
		projectName: cfg.ProjectName,
		domain:      cfg.Domain,
		workDir:     cfg.WorkDir,
		useHashing:  useHashing,
		docker:      cfg.Docker,
		cache:       newCache(cfg.CacheTTL, 1000),
		logger:      cfg.Logger,
	}

	// Create DNS server
	mux := dns.NewServeMux()
	mux.HandleFunc(cfg.Domain+".", s.handleOrbLocal)
	mux.HandleFunc(".", s.handleUpstream)

	s.server = &dns.Server{
		Addr:    cfg.Addr,
		Net:     "udp",
		Handler: mux,
	}

	return s, nil
}

// Start starts the DNS server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Starting DNS server", "addr", s.addr, "domain", s.domain)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	// Wait a bit to ensure server started
	select {
	case err := <-errCh:
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return fmt.Errorf("failed to start DNS server: %w", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully
		s.logger.Info("DNS server started successfully")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stop stops the DNS server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping DNS server")

	if err := s.server.Shutdown(); err != nil {
		return fmt.Errorf("failed to stop DNS server: %w", err)
	}

	s.running = false
	s.logger.Info("DNS server stopped")
	return nil
}

// handleOrbLocal handles *.orb.local queries
func (s *Server) handleOrbLocal(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		if q.Qtype != dns.TypeA {
			continue
		}

		hostname := strings.TrimSuffix(q.Name, ".")

		// Check cache first
		if ip := s.cache.get(hostname); ip != "" {
			s.logger.Debug("DNS cache hit", "hostname", hostname, "ip", ip)
			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    30,
				},
				A: net.ParseIP(ip),
			}
			m.Answer = append(m.Answer, rr)
			continue
		}

		// Resolve from Docker
		ip, err := s.resolveContainerIP(context.Background(), hostname)
		if err != nil {
			s.logger.Warn("Failed to resolve container", "hostname", hostname, "error", err)
			continue
		}

		if ip != "" {
			// Cache the result
			s.cache.set(hostname, ip)

			s.logger.Debug("DNS resolved", "hostname", hostname, "ip", ip)

			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    30,
				},
				A: net.ParseIP(ip),
			}
			m.Answer = append(m.Answer, rr)
		}
	}

	if err := w.WriteMsg(m); err != nil {
		s.logger.Debug("Failed to write DNS response", "error", err)
	}
}

// handleUpstream forwards queries to upstream DNS
func (s *Server) handleUpstream(w dns.ResponseWriter, r *dns.Msg) {
	// Forward to upstream DNS
	c := new(dns.Client)
	c.Timeout = 2 * time.Second

	resp, _, err := c.Exchange(r, s.upstream)
	if err != nil {
		s.logger.Warn("Failed to forward DNS query", "error", err)
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		if err := w.WriteMsg(m); err != nil {
			s.logger.Debug("Failed to write DNS error response", "error", err)
		}
		return
	}

	if err := w.WriteMsg(resp); err != nil {
		s.logger.Debug("Failed to write DNS response", "error", err)
	}
}

// resolveContainerIP resolves a container IP from its hostname
func (s *Server) resolveContainerIP(ctx context.Context, hostname string) (string, error) {
	// Strip domain suffix
	hostname = strings.TrimSuffix(hostname, "."+s.domain)
	hostname = strings.TrimSuffix(hostname, ".")

	// Check if this is a valid hashed domain (service-hash.domain)
	fullHostname := hostname + "." + s.domain
	if ValidateHashedDomain(fullHostname, s.domain) {
		// Extract service name and hash from hashed domain
		serviceName := ExtractServiceNameFromHashedDomain(fullHostname, s.domain)
		hash := ExtractHashFromHashedDomain(fullHostname, s.domain)

		s.logger.Debug("Extracted service and hash from domain",
			"hostname", hostname,
			"service", serviceName,
			"hash", hash)

		// Look up container by both service name AND hash
		ip, err := s.docker.GetContainerIPByHash(ctx, serviceName, hash)
		if err != nil {
			s.logger.Warn("Failed to resolve container by hash",
				"service", serviceName,
				"hash", hash,
				"error", err)
			return "", err
		}

		return ip, nil
	}

	// Not a hashed domain - fall back to legacy lookup by service name only
	s.logger.Debug("Non-hashed domain lookup", "hostname", hostname)
	ip, err := s.docker.GetContainerIP(ctx, s.projectName, hostname)
	if err != nil {
		return "", err
	}

	return ip, nil
}

// IsRunning returns true if the server is running
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.addr
}
