package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kevin-cantwell/expose/internal/state"
)

func writeState(s state.TunnelState) (cleanup func(), err error) {
	dir, err := tunnelsDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating tunnels dir: %w", err)
	}
	data, _ := json.Marshal(s)
	path := filepath.Join(dir, s.Subdomain+".json")
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
