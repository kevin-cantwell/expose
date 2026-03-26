package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// RestartCmd stops and restarts a background tunnel with the same parameters.
type RestartCmd struct {
	Subdomain string `arg:"" help:"Subdomain of the tunnel to restart"`
	Server    string `help:"Expose server domain" env:"EXPOSE_SERVER"`
}

func (c *RestartCmd) Run() error {
	s, err := ReadState(c.Subdomain)
	if err != nil {
		return err
	}

	server := c.Server
	if server == "" {
		server = os.Getenv("EXPOSE_SERVER")
	}
	if server == "" && s.PublicURL != "" {
		// Derive server from the saved public URL (https://<subdomain>.<server>)
		if u, err := url.Parse(s.PublicURL); err == nil {
			server = strings.TrimPrefix(u.Host, s.Subdomain+".")
		}
	}
	if server == "" {
		return fmt.Errorf("server domain required: set --server or EXPOSE_SERVER env var")
	}

	if err := (&StopCmd{Subdomain: c.Subdomain}).Run(); err != nil {
		return fmt.Errorf("stopping tunnel: %w", err)
	}

	return (&StartCmd{
		Addr:      s.LocalAddr,
		Subdomain: c.Subdomain,
		Server:    server,
	}).Run()
}
