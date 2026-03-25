package cmd

import (
	"github.com/kevin-cantwell/expose/server"
)

// ServeCmd runs the expose server on the DO droplet.
type ServeCmd struct {
	Domain      string `help:"Base domain for tunnels" env:"EXPOSE_DOMAIN" required:""`
	AllowedUser string `help:"GitHub username allowed to connect" env:"EXPOSE_ALLOWED_USER" required:""`
	CertDir     string `help:"Directory to store TLS certificates" default:"/var/lib/expose/certs" env:"EXPOSE_CERT_DIR"`
	Email       string `help:"Email for Let's Encrypt registration" env:"EXPOSE_ACME_EMAIL"`
	Staging     bool   `help:"Use Let's Encrypt staging environment" env:"EXPOSE_STAGING"`
}

func (c *ServeCmd) Run() error {
	s := server.New(server.Config{
		Domain:      c.Domain,
		AllowedUser: c.AllowedUser,
		CertDir:     c.CertDir,
		Email:       c.Email,
		Staging:     c.Staging,
	})
	return s.Start()
}
