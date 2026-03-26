package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/kevin-cantwell/expose/internal/state"
)

// ReadState reads the state file for a given subdomain.
func ReadState(subdomain string) (state.TunnelState, error) {
	dir, err := TunnelsDir()
	if err != nil {
		return state.TunnelState{}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, subdomain+".json"))
	if err != nil {
		return state.TunnelState{}, fmt.Errorf("no active tunnel named %q", subdomain)
	}
	var s state.TunnelState
	if err := json.Unmarshal(data, &s); err != nil {
		return state.TunnelState{}, fmt.Errorf("corrupt state for %q: %w", subdomain, err)
	}
	return s, nil
}

// RemoveState deletes the state file for a given subdomain.
func RemoveState(subdomain string) {
	dir, err := TunnelsDir()
	if err != nil {
		return
	}
	os.Remove(filepath.Join(dir, subdomain+".json"))
}

// IsProcessAlive returns true if a process with the given PID is running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// LogsDir returns the path to the tunnel log directory.
func LogsDir() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "logs"), nil
}

// LogFilePath returns the log file path for a given subdomain.
func LogFilePath(subdomain string) (string, error) {
	dir, err := LogsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, subdomain+".log"), nil
}
