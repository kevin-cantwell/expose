package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// LogsCmd displays logs for a tunnel.
type LogsCmd struct {
	Subdomain string `arg:"" help:"Subdomain of the tunnel"`
	Follow    bool   `short:"f" help:"Follow log output (like tail -f)"`
}

func (c *LogsCmd) Run() error {
	logFile, err := c.resolveLogFile()
	if err != nil {
		return err
	}
	if c.Follow {
		cmd := exec.Command("tail", "-f", logFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	f, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()
	_, err = io.Copy(os.Stdout, f)
	return err
}

func (c *LogsCmd) resolveLogFile() (string, error) {
	// Try to get the log path from the running state file first
	s, err := ReadState(c.Subdomain)
	if err == nil {
		if s.LogFile != "" {
			return s.LogFile, nil
		}
		return "", fmt.Errorf("tunnel %q is a foreground tunnel with no log file", c.Subdomain)
	}
	// Tunnel may have exited; try the canonical log path for post-mortem inspection
	logFile, err := LogFilePath(c.Subdomain)
	if err != nil {
		return "", fmt.Errorf("could not determine log path for %q", c.Subdomain)
	}
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return "", fmt.Errorf("no log file found for %q", c.Subdomain)
	}
	return logFile, nil
}
