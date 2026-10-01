package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	words "github.com/kevin-cantwell/expose/internal"
)

// StartCmd starts a tunnel as a detached background process.
type StartCmd struct {
	Addr      string `arg:"" help:"Local address to tunnel (e.g. :3000 or localhost:3000)"`
	Subdomain string `short:"s" help:"Custom subdomain (auto-generated if omitted)"`
	Server    string `help:"Expose server domain" env:"EXPOSE_SERVER"`
}

func (c *StartCmd) Run() error {
	server := c.Server
	if server == "" {
		server = os.Getenv("EXPOSE_SERVER")
	}
	if server == "" {
		return fmt.Errorf("server domain required: set --server or EXPOSE_SERVER env var")
	}

	subdomain := c.Subdomain
	if subdomain == "" {
		subdomain = words.Random()
	}

	logFile, err := LogFilePath(subdomain)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(logFile), 0700); err != nil {
		return fmt.Errorf("creating logs dir: %w", err)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	cmd := exec.Command(exe,
		c.Addr,
		"--subdomain", subdomain,
		"--server", server,
		"--background",
		"--log-file", logFile,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting background tunnel: %w", err)
	}

	// Wait until the child either connects (writes its state file) or exits,
	// so failures surface here instead of leaving a silently retrying process.
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	deadline := time.After(10 * time.Second)
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	connected := false
wait:
	for {
		select {
		case <-exited:
			return fmt.Errorf("background tunnel exited:\n%s", tailFile(logFile, 5))
		case <-tick.C:
			if s, err := ReadState(subdomain); err == nil && s.PID == cmd.Process.Pid {
				connected = true
				break wait
			}
		case <-deadline:
			break wait
		}
	}

	publicURL := "https://" + subdomain + "." + server
	if !connected {
		fmt.Printf("Warning: tunnel hasn't connected yet and is still retrying:\n%s\n\n", tailFile(logFile, 5))
	}
	fmt.Printf("Started background tunnel\n\n")
	fmt.Printf("  Subdomain: %s\n", subdomain)
	fmt.Printf("  URL:       %s\n", publicURL)
	fmt.Printf("  Logs:      %s\n", logFile)
	fmt.Printf("  PID:       %d\n\n", cmd.Process.Pid)
	fmt.Printf("Use 'expose logs %s -f' to follow logs\n", subdomain)
	fmt.Printf("Use 'expose stop %s' to stop\n", subdomain)
	return nil
}

// tailFile returns the last n non-empty lines of a file, indented for display.
func tailFile(path string, n int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "  (no log output)"
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return "  " + strings.Join(lines, "\n  ")
}
