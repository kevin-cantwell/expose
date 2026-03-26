package server

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/yamux"
)

// Tunnel represents a connected client tunnel session.
type Tunnel struct {
	Subdomain string
	LocalAddr string
	session   *yamux.Session
	connectedAt time.Time
}

func newTunnel(subdomain, localAddr string, session *yamux.Session) *Tunnel {
	return &Tunnel{
		Subdomain:   subdomain,
		LocalAddr:   localAddr,
		session:     session,
		connectedAt: time.Now(),
	}
}

// Close shuts down the underlying yamux session.
func (t *Tunnel) Close() {
	t.session.Close()
}

// IsClosed reports whether the session has been closed.
func (t *Tunnel) IsClosed() bool {
	return t.session.IsClosed()
}

// ServeHTTP proxies an incoming HTTP request through the tunnel to the client.
func (t *Tunnel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Detect WebSocket upgrade requests and handle them specially.
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		t.serveWebSocket(w, r)
		return
	}

	stream, err := t.session.Open()
	if err != nil {
		http.Error(w, fmt.Sprintf("tunnel unavailable: %v", err), http.StatusBadGateway)
		return
	}
	defer stream.Close()

	// Forward the raw HTTP request
	if err := r.Write(stream); err != nil {
		http.Error(w, "failed to forward request", http.StatusBadGateway)
		return
	}

	// Read the raw HTTP response
	resp, err := http.ReadResponse(bufio.NewReader(stream), r)
	if err != nil {
		http.Error(w, "failed to read response from tunnel", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// serveWebSocket handles WebSocket upgrade requests by forwarding the upgrade
// through the tunnel and then piping both connections bidirectionally.
func (t *Tunnel) serveWebSocket(w http.ResponseWriter, r *http.Request) {
	// Open a yamux stream for this WebSocket connection.
	stream, err := t.session.Open()
	if err != nil {
		http.Error(w, fmt.Sprintf("tunnel unavailable: %v", err), http.StatusBadGateway)
		return
	}
	// stream will be closed after the pipe goroutines finish.

	// Forward the raw HTTP upgrade request to the client-side tunnel.
	if err := r.Write(stream); err != nil {
		stream.Close()
		http.Error(w, "failed to forward WebSocket upgrade request", http.StatusBadGateway)
		return
	}

	// Read the response from the local server (via the client tunnel).
	br := bufio.NewReader(stream)
	resp, err := http.ReadResponse(br, r)
	if err != nil {
		stream.Close()
		http.Error(w, "failed to read WebSocket upgrade response", http.StatusBadGateway)
		return
	}

	// If the local server did not agree to upgrade, proxy the response normally.
	if resp.StatusCode != http.StatusSwitchingProtocols {
		defer resp.Body.Close()
		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		stream.Close()
		return
	}
	resp.Body.Close()

	// Hijack the client connection so we can speak raw TCP.
	hj, ok := w.(http.Hijacker)
	if !ok {
		stream.Close()
		http.Error(w, "WebSocket not supported: server does not implement http.Hijacker", http.StatusInternalServerError)
		return
	}
	clientConn, clientBuf, err := hj.Hijack()
	if err != nil {
		stream.Close()
		return
	}

	// Send the 101 Switching Protocols response to the browser.
	if err := resp.Write(clientConn); err != nil {
		clientConn.Close()
		stream.Close()
		return
	}

	// Flush any bytes the bufio.Reader already buffered from the stream side.
	var streamLeadBytes []byte
	if br.Buffered() > 0 {
		streamLeadBytes = make([]byte, br.Buffered())
		io.ReadFull(br, streamLeadBytes)
	}

	// Pipe both directions concurrently. Close both ends when either side is done.
	done := make(chan struct{}, 2)

	go func() {
		if len(streamLeadBytes) > 0 {
			clientConn.Write(streamLeadBytes)
		}
		io.Copy(clientConn, stream)
		done <- struct{}{}
	}()
	go func() {
		// clientBuf may have data already read from the browser.
		if clientBuf.Reader.Buffered() > 0 {
			buf := make([]byte, clientBuf.Reader.Buffered())
			clientBuf.Reader.Read(buf)
			stream.Write(buf)
		}
		io.Copy(stream, clientConn)
		done <- struct{}{}
	}()

	<-done
	clientConn.Close()
	stream.Close()
	<-done // drain second goroutine
}
