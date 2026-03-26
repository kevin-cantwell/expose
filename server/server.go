package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/hashicorp/yamux"
)

var validSubdomain = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)

// Config holds server configuration.
type Config struct {
	Domain      string
	AllowedUser string
	CertDir     string
	Email       string
	Staging     bool
}

// Server handles tunnel connections and proxies HTTP requests.
type Server struct {
	cfg     Config
	tunnels sync.Map // subdomain → *Tunnel
}

// New creates a new Server.
func New(cfg Config) *Server {
	return &Server{cfg: cfg}
}

// Start sets up TLS, starts listeners on :443 and :80, and blocks.
func (s *Server) Start() error {
	tlsCfg, err := buildTLSConfig(s.cfg.Domain, s.cfg.CertDir, s.cfg.Email, s.cfg.Staging)
	if err != nil {
		return fmt.Errorf("TLS setup: %w", err)
	}

	// HTTP → HTTPS redirect on :80
	go func() {
		log.Println("Listening on :80 (redirect)")
		if err := http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		})); err != nil {
			log.Printf("HTTP redirect listener: %v", err)
		}
	}()

	// Ensure ALPN includes h2 and http/1.1 so browsers don't fail the handshake.
	tlsCfg.NextProtos = append(tlsCfg.NextProtos, "h2", "http/1.1")

	ln := tls.NewListener(mustListen(":443"), tlsCfg)
	log.Printf("Listening on :443 (domain: %s)", s.cfg.Domain)

	srv := &http.Server{Handler: s, TLSConfig: tlsCfg}
	return srv.Serve(ln)
}

func mustListen(addr string) net.Listener {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}
	return ln
}

// ServeHTTP routes requests by subdomain.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := stripPort(r.Host)
	sub := strings.TrimSuffix(host, "."+s.cfg.Domain)

	// Requests to the apex or www — nothing running there
	if sub == host || sub == "" || sub == "www" {
		http.NotFound(w, r)
		return
	}

	// "expose" subdomain is the control plane
	if sub == "expose" {
		if r.URL.Path == "/connect" {
			s.handleConnect(w, r)
			return
		}
		http.NotFound(w, r)
		return
	}

	// Look up tunnel for subdomain
	val, ok := s.tunnels.Load(sub)
	if !ok {
		http.Error(w, fmt.Sprintf("no tunnel for %s.%s", sub, s.cfg.Domain), http.StatusBadGateway)
		return
	}
	val.(*Tunnel).ServeHTTP(w, r)
}

// handleConnect upgrades an HTTP request to a WebSocket + yamux tunnel session.
func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	// Authenticate via GitHub token
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		http.Error(w, "missing Authorization header", http.StatusUnauthorized)
		return
	}
	username, err := verifyGitHubToken(token)
	if err != nil {
		log.Printf("auth failed: %v", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if username != s.cfg.AllowedUser {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	subdomain := r.Header.Get("X-Subdomain")
	localAddr := r.Header.Get("X-Local-Addr")
	if !validSubdomain.MatchString(subdomain) || subdomain == "expose" {
		http.Error(w, "invalid subdomain: must match ^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$", http.StatusBadRequest)
		return
	}

	// Reserve the subdomain atomically
	if _, loaded := s.tunnels.LoadOrStore(subdomain, (*Tunnel)(nil)); loaded {
		http.Error(w, fmt.Sprintf("subdomain %q already in use", subdomain), http.StatusConflict)
		return
	}

	// Upgrade to WebSocket
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		s.tunnels.Delete(subdomain)
		return
	}

	ctx := context.Background()
	netConn := websocket.NetConn(ctx, conn, websocket.MessageBinary)
	session, err := yamux.Client(netConn, yamuxConfig())
	if err != nil {
		conn.Close(websocket.StatusInternalError, "yamux init failed")
		s.tunnels.Delete(subdomain)
		return
	}

	tunnel := newTunnel(subdomain, localAddr, session)
	s.tunnels.Store(subdomain, tunnel)

	publicURL := "https://" + subdomain + "." + s.cfg.Domain
	log.Printf("tunnel connected: %s → %s (%s)", publicURL, localAddr, username)

	// Block until session closes
	for !session.IsClosed() {
		time.Sleep(500 * time.Millisecond)
	}

	s.tunnels.Delete(subdomain)
	log.Printf("tunnel disconnected: %s", publicURL)
}

func yamuxConfig() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = 15 * time.Second
	cfg.ConnectionWriteTimeout = 10 * time.Second
	return cfg
}

func stripPort(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return host
}
