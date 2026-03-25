package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/hashicorp/yamux"
	words "github.com/kevin-cantwell/expose/internal"
)

// Config holds client configuration.
type Config struct {
	LocalAddr string // e.g. ":3000" or "localhost:3000"
	Subdomain string // empty → auto-generate
	Server    string // e.g. "example.com" — set via EXPOSE_SERVER
	Token     string // GitHub OAuth token
}

// Client manages the tunnel to the expose server.
type Client struct {
	cfg Config
	tui *TUI
}

// New creates a new Client.
func New(cfg Config) *Client {
	// Normalize local addr
	if cfg.LocalAddr != "" && !strings.Contains(cfg.LocalAddr, ":") {
		cfg.LocalAddr = ":" + cfg.LocalAddr
	}
	return &Client{cfg: cfg, tui: newTUI()}
}

// Run connects to the server and starts tunneling, reconnecting on failure.
func (c *Client) Run() error {
	subdomain := c.cfg.Subdomain
	if subdomain == "" {
		subdomain = words.Random()
	}

	backoff := time.Second
	first := true

	for {
		if !first {
			PrintStatus(fmt.Sprintf("reconnecting in %s...", backoff))
			time.Sleep(backoff)
			if backoff < 30*time.Second {
			backoff *= 2
		}
		}
		first = false

		if err := c.connect(subdomain); err != nil {
			PrintStatus(fmt.Sprintf("disconnected: %v", err))
			continue
		}
		// Clean reconnect — reset backoff
		backoff = time.Second
	}
}

func (c *Client) connect(subdomain string) error {
	serverURL := fmt.Sprintf("wss://expose.%s/connect", c.cfg.Server)

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, serverURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": {"Bearer " + c.cfg.Token},
			"X-Subdomain":   {subdomain},
			"X-Local-Addr":  {c.cfg.LocalAddr},
		},
	})
	if err != nil {
		return fmt.Errorf("connecting to server: %w", err)
	}
	defer conn.CloseNow()

	netConn := websocket.NetConn(ctx, conn, websocket.MessageBinary)
	session, err := yamux.Server(netConn, yamuxConfig())
	if err != nil {
		return fmt.Errorf("yamux init: %w", err)
	}
	defer session.Close()

	publicURL := "https://" + subdomain + "." + c.cfg.Server

	// Write state file for `expose ls`
	cleanup, err := writeState(subdomain, publicURL, c.cfg.LocalAddr)
	if err != nil {
		PrintStatus(fmt.Sprintf("warning: couldn't write state file: %v", err))
	} else {
		defer cleanup()
	}

	c.tui.Start(publicURL, c.cfg.LocalAddr)

	// Accept yamux streams — each is one HTTP request from the server
	for {
		stream, err := session.Accept()
		if err != nil {
			return err
		}
		go c.handleStream(stream)
	}
}

func (c *Client) handleStream(stream net.Conn) {
	defer stream.Close()

	start := time.Now()

	// Read the HTTP request forwarded from the server
	req, err := http.ReadRequest(bufio.NewReader(stream))
	if err != nil {
		return
	}

	// Dial the local service
	localAddr := c.cfg.LocalAddr
	if strings.HasPrefix(localAddr, ":") {
		localAddr = "localhost" + localAddr
	}
	local, err := net.DialTimeout("tcp", localAddr, 10*time.Second)
	if err != nil {
		writeErrorResponse(stream, http.StatusBadGateway,
			fmt.Sprintf("local service unavailable: %v", err))
		c.tui.Log(RequestLog{
			Time:     start,
			Method:   req.Method,
			Path:     req.URL.Path,
			Status:   http.StatusBadGateway,
			Duration: time.Since(start),
		})
		return
	}
	defer local.Close()

	// Forward the request to the local service
	if err := req.Write(local); err != nil {
		return
	}

	// Read the response from local and write back to the stream
	resp, err := http.ReadResponse(bufio.NewReader(local), req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if err := resp.Write(stream); err != nil {
		return
	}

	c.tui.Log(RequestLog{
		Time:     start,
		Method:   req.Method,
		Path:     req.URL.Path,
		Status:   resp.StatusCode,
		Duration: time.Since(start),
	})
}

func writeErrorResponse(w io.Writer, status int, msg string) {
	body := msg
	fmt.Fprintf(w, "HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		status, http.StatusText(status), len(body), body)
}

func yamuxConfig() *yamux.Config {
	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = 15 * time.Second
	cfg.ConnectionWriteTimeout = 10 * time.Second
	return cfg
}

