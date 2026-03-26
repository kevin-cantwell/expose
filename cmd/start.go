package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

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

	publicURL := "https://" + subdomain + "." + server
	fmt.Printf("Started background tunnel\n\n")
	fmt.Printf("  Subdomain: %s\n", subdomain)
	fmt.Printf("  URL:       %s\n", publicURL)
	fmt.Printf("  Logs:      %s\n", logFile)
	fmt.Printf("  PID:       %d\n\n", cmd.Process.Pid)
	fmt.Printf("Use 'expose logs %s -f' to follow logs\n", subdomain)
	fmt.Printf("Use 'expose stop %s' to stop\n", subdomain)
	return nil
}
