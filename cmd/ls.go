package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/kevin-cantwell/expose/internal/state"
)

// LsCmd lists active tunnels on this machine by reading state files.
type LsCmd struct{}

func (c *LsCmd) Run() error {
	dir, err := TunnelsDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		fmt.Println("No active tunnels.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading tunnels dir: %w", err)
	}

	var active []state.TunnelState
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var s state.TunnelState
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		// Check if the process is still alive
		if !isProcessAlive(s.PID) {
			// Clean up stale state file
			os.Remove(filepath.Join(dir, entry.Name()))
			continue
		}
		active = append(active, s)
	}

	if len(active) == 0 {
		fmt.Println("No active tunnels.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SUBDOMAIN\tURL\tLOCAL\tSINCE")
	for _, s := range active {
		since := time.Since(s.StartedAt).Truncate(time.Second)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Subdomain, s.PublicURL, s.LocalAddr, since)
	}
	return w.Flush()
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds; send signal 0 to check existence
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
