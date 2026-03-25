package server

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
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
