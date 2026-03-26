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

		err := c.connect(subdomain)
		if err == nil {
			backoff = time.Second
			continue
		}
		// Don't retry on permanent errors
		if fatal, ok := err.(*fatalError); ok {
			return fatal.err
		}
		PrintStatus(fmt.Sprintf("disconnected: %v", err))
	}
}

// fatalError wraps errors that should not trigger reconnection.
type fatalError struct{ err error }

func (e *fatalError) Error() string { return e.err.Error() }

func (c *Client) connect(subdomain string) error {
	// Pre-flight check: warn if nothing is listening on the local port yet
	localCheck := c.cfg.LocalAddr
	if strings.HasPrefix(localCheck, ":") {
		localCheck = "localhost" + localCheck
	}
	if tc, err := net.DialTimeout("tcp", localCheck, 500*time.Millisecond); err != nil {
		fmt.Printf("warning: nothing listening on %s yet — connecting anyway\n", c.cfg.LocalAddr)
	} else {
		tc.Close()
	}

	serverURL := fmt.Sprintf("wss://expose.%s/connect", c.cfg.Server)

	ctx := context.Background()
	conn, resp, err := websocket.Dial(ctx, serverURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Authorization": {"Bearer " + c.cfg.Token},
			"X-Subdomain":   {subdomain},
			"X-Local-Addr":  {c.cfg.LocalAddr},
		},
	})
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			msg := strings.TrimSpace(string(body))
			switch resp.StatusCode {
			case http.StatusConflict:
				return &fatalError{fmt.Errorf("%s.%s is already in use", subdomain, c.cfg.Server)}
			case http.StatusUnauthorized:
				return &fatalError{fmt.Errorf("authentication failed: run 'expose login'")}
			case http.StatusForbidden:
				return &fatalError{fmt.Errorf("access denied: GitHub user not allowed")}
			default:
				if msg != "" {
					return fmt.Errorf("server error %d: %s", resp.StatusCode, msg)
				}
			}
		}
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
	br := bufio.NewReader(stream)
	req, err := http.ReadRequest(br)
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
			Path:     req.URL.RequestURI(),
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

	// Read the response from local
	localBr := bufio.NewReader(local)
	resp, err := http.ReadResponse(localBr, req)
	if err != nil {
		return
	}

	// WebSocket upgrade: pipe both connections bidirectionally after the 101.
	if resp.StatusCode == http.StatusSwitchingProtocols {
		resp.Body.Close()

		// Write the 101 response back to the server-side tunnel stream.
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

		// Flush any bytes already buffered from each side.
		var streamLeadBytes, localLeadBytes []byte
		if br.Buffered() > 0 {
			streamLeadBytes = make([]byte, br.Buffered())
			io.ReadFull(br, streamLeadBytes)
		}
		if localBr.Buffered() > 0 {
			localLeadBytes = make([]byte, localBr.Buffered())
			io.ReadFull(localBr, localLeadBytes)
		}

		done := make(chan struct{}, 2)
		go func() {
			if len(localLeadBytes) > 0 {
				stream.Write(localLeadBytes)
			}
			io.Copy(stream, local)
			done <- struct{}{}
		}()
		go func() {
			if len(streamLeadBytes) > 0 {
				local.Write(streamLeadBytes)
			}
			io.Copy(local, stream)
			done <- struct{}{}
		}()
		<-done
		// Close both ends to unblock the other goroutine.
		stream.Close()
		local.Close()
		<-done
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

