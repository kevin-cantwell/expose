package cmd

import (
	"github.com/kevin-cantwell/expose/client"
)

// TunnelCmd is the default command: expose :3000 [--subdomain foo]
type TunnelCmd struct {
	Addr      string `arg:"" help:"Local address to tunnel (e.g. :3000 or localhost:3000)"`
	Subdomain string `short:"s" help:"Custom subdomain (auto-generated if omitted)"`
	Server    string `help:"Expose server domain" env:"EXPOSE_SERVER" required:""`
}

func (c *TunnelCmd) Run() error {
	token, err := LoadToken()
	if err != nil {
		return err
	}

	cl := client.New(client.Config{
		LocalAddr: c.Addr,
		Subdomain: c.Subdomain,
		Server:    c.Server,
		Token:     token,
	})
	return cl.Run()
}
