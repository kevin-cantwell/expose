package cmd

import (
	"fmt"
	"os"

	"github.com/kevin-cantwell/expose/client"
)

// TunnelCmd is the default command: expose :3000 [--subdomain foo]
type TunnelCmd struct {
	Addr       string `arg:"" help:"Local address to tunnel (e.g. :3000 or localhost:3000)"`
	Subdomain  string `short:"s" help:"Custom subdomain (auto-generated if omitted)"`
	Server     string `help:"Expose server domain" env:"EXPOSE_SERVER"`
	Background bool   `name:"background" help:"-" hidden:""`
	LogFile    string `name:"log-file"   help:"-" hidden:""`
}

func (c *TunnelCmd) Run() error {
	if c.Server == "" {
		c.Server = os.Getenv("EXPOSE_SERVER")
	}
	if c.Server == "" {
		return fmt.Errorf("server domain required: set --server or EXPOSE_SERVER env var")
	}
	token, err := LoadToken()
	if err != nil {
		return err
	}

	cl, err := client.New(client.Config{
		LocalAddr:  c.Addr,
		Subdomain:  c.Subdomain,
		Server:     c.Server,
		Token:      token,
		Background: c.Background,
		LogFile:    c.LogFile,
	})
	if err != nil {
		return err
	}
	return cl.Run()
}
