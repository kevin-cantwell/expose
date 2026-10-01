package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/digitalocean"
)

// buildTLSConfig sets up certmagic with the DigitalOcean DNS-01 solver
// and returns a *tls.Config for wildcard certs on *.domain.
func buildTLSConfig(domain, certDir, email string, staging bool) (*tls.Config, error) {
	doToken := os.Getenv("DO_AUTH_TOKEN")
	if doToken == "" {
		return nil, fmt.Errorf("DO_AUTH_TOKEN env var required for DNS-01 challenge")
	}

	ca := certmagic.LetsEncryptProductionCA
	if staging {
		ca = certmagic.LetsEncryptStagingCA
	}

	// Use a dedicated cache whose GetConfigForCert returns our config.
	// certmagic.NewDefault() would hand background renewals certmagic.Default,
	// which has no DNS-01 solver — so the wildcard cert could never renew.
	var magic *certmagic.Config
	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(certmagic.Certificate) (*certmagic.Config, error) {
			return magic, nil
		},
	})
	magic = certmagic.New(cache, certmagic.Config{
		Storage: &certmagic.FileStorage{Path: certDir},
	})
	magic.Issuers = []certmagic.Issuer{
		certmagic.NewACMEIssuer(magic, certmagic.ACMEIssuer{
			CA:     ca,
			Email:  email,
			Agreed: true,
			DNS01Solver: &certmagic.DNS01Solver{
				DNSManager: certmagic.DNSManager{
					DNSProvider: &digitalocean.Provider{APIToken: doToken},
				},
			},
			DisableHTTPChallenge:    true,
			DisableTLSALPNChallenge: true,
		}),
	}

	// Obtain certificates for the wildcard and the apex domain
	domains := []string{"*." + domain}
	if err := magic.ManageSync(context.Background(), domains); err != nil {
		return nil, fmt.Errorf("obtaining TLS cert for %v: %w", domains, err)
	}

	return magic.TLSConfig(), nil
}
