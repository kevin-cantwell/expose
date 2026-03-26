package state

import "time"

// TunnelState is the on-disk format for a running tunnel.
type TunnelState struct {
	Subdomain  string    `json:"subdomain"`
	PublicURL  string    `json:"public_url"`
	LocalAddr  string    `json:"local_addr"`
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"started_at"`
	Background bool      `json:"background,omitempty"`
	LogFile    string    `json:"log_file,omitempty"`
}
