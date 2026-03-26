package cmd

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

// StopCmd stops a running tunnel by subdomain.
type StopCmd struct {
	Subdomain string `arg:"" help:"Subdomain of the tunnel to stop"`
}

func (c *StopCmd) Run() error {
	s, err := ReadState(c.Subdomain)
	if err != nil {
		return err
	}

	proc, err := os.FindProcess(s.PID)
	if err != nil {
		RemoveState(c.Subdomain)
		return fmt.Errorf("finding process %d: %w", s.PID, err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		if errors.Is(err, os.ErrProcessDone) {
			fmt.Printf("Process %d already exited\n", s.PID)
			RemoveState(c.Subdomain)
			return nil
		}
		return fmt.Errorf("stopping process %d: %w", s.PID, err)
	}

	// Wait up to 5s for clean exit before forcing
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
		if !IsProcessAlive(s.PID) {
			break
		}
	}
	if IsProcessAlive(s.PID) {
		fmt.Printf("Process %d did not exit after 5s, sending SIGKILL\n", s.PID)
		proc.Signal(syscall.SIGKILL)
	}

	// Remove state file — the tunnel process won't clean it up when killed by signal
	RemoveState(c.Subdomain)
	fmt.Printf("Stopped tunnel %s (PID %d)\n", c.Subdomain, s.PID)
	return nil
}
