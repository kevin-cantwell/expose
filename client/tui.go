package client

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// RequestLog holds metadata for a single proxied request.
type RequestLog struct {
	Time     time.Time
	Method   string
	Path     string
	Status   int
	Duration time.Duration
}

// TUI manages terminal output for the expose client.
type TUI struct {
	w        io.Writer
	plain    bool // plain text mode for log files (no ANSI, with timestamps)
	requests chan RequestLog
	done     chan struct{}
}

func newTUI(w io.Writer, plain bool) *TUI {
	return &TUI{
		w:        w,
		plain:    plain,
		requests: make(chan RequestLog, 64),
		done:     make(chan struct{}),
	}
}

// Start prints the connection banner and starts the request log loop.
func (t *TUI) Start(publicURL, localAddr string) {
	if t.plain {
		fmt.Fprintf(t.w, "%s expose connected\n", time.Now().Format(time.RFC3339))
		fmt.Fprintf(t.w, "  URL:   %s\n", publicURL)
		fmt.Fprintf(t.w, "  Local: http://%s\n\n", displayAddr(localAddr))
	} else {
		fmt.Fprintf(t.w, "\n%sexpose%s connected\n\n", colorCyan, colorReset)
		fmt.Fprintf(t.w, "  URL:    %s%s%s\n", colorGreen, publicURL, colorReset)
		fmt.Fprintf(t.w, "  Local:  http://%s\n", displayAddr(localAddr))
		fmt.Fprintf(t.w, "  Time:   %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Fprintf(t.w, "%s%-6s  %-40s  %s  %s%s\n", colorGray, "METHOD", "PATH", "STATUS", "LATENCY", colorReset)
		fmt.Fprintf(t.w, "%s%s%s\n", colorGray, repeat("─", 65), colorReset)
	}
	go t.loop()
}

// Log records an incoming request for display.
func (t *TUI) Log(r RequestLog) {
	select {
	case t.requests <- r:
	default:
	}
}

// Stop shuts down the TUI loop.
func (t *TUI) Stop() {
	close(t.done)
}

// PrintStatus prints a status update (reconnecting, warnings, etc).
func (t *TUI) PrintStatus(msg string) {
	if t.plain {
		fmt.Fprintf(t.w, "%s %s\n", time.Now().Format(time.RFC3339), msg)
	} else {
		fmt.Fprintf(t.w, "%s[%s] %s%s\n", colorYellow, time.Now().Format("15:04:05"), msg, colorReset)
	}
}

func (t *TUI) loop() {
	for {
		select {
		case <-t.done:
			return
		case r := <-t.requests:
			if t.plain {
				fmt.Fprintf(t.w, "%s %-6s %s %d %s\n",
					r.Time.Format(time.RFC3339),
					r.Method, r.Path, r.Status,
					r.Duration.Round(time.Millisecond),
				)
			} else {
				fmt.Fprintf(t.w, "%s%-6s%s  %-40s  %s%d%s  %s\n",
					colorForMethod(r.Method), r.Method, colorReset,
					truncate(r.Path, 40),
					statusColor(r.Status), r.Status, colorReset,
					r.Duration.Round(time.Millisecond),
				)
			}
		}
	}
}

func statusColor(status int) string {
	switch {
	case status >= 500:
		return colorRed
	case status >= 400:
		return colorYellow
	default:
		return colorGreen
	}
}

func colorForMethod(method string) string {
	switch method {
	case http.MethodGet:
		return colorCyan
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return colorGreen
	case http.MethodDelete:
		return colorRed
	default:
		return colorReset
	}
}

func displayAddr(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func repeat(s string, n int) string {
	b := make([]byte, n*len(s))
	for i := range n {
		copy(b[i*len(s):], s)
	}
	return string(b)
}
