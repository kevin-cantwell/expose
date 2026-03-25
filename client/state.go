package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type tunnelState struct {
	Subdomain string    `json:"subdomain"`
	PublicURL string    `json:"public_url"`
	LocalAddr string    `json:"local_addr"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
}

func writeState(sub, publicURL, localAddr string) (cleanup func(), err error) {
	dir, err := tunnelsDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating tunnels dir: %w", err)
	}

	s := tunnelState{
		Subdomain: sub,
		PublicURL: publicURL,
		LocalAddr: localAddr,
		PID:       os.Getpid(),
		StartedAt: time.Now(),
	}
	data, _ := json.Marshal(s)
	path := filepath.Join(dir, sub+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, fmt.Errorf("writing state: %w", err)
	}

	return func() { os.Remove(path) }, nil
}

func tunnelsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "expose", "tunnels"), nil
}
