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

	certmagic.DefaultACME.Agreed = true
	if email != "" {
		certmagic.DefaultACME.Email = email
	}
	if staging {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	}

	certmagic.Default.Storage = &certmagic.FileStorage{Path: certDir}

	dnsProvider := &digitalocean.Provider{
		APIToken: doToken,
	}

	magic := certmagic.NewDefault()
	magic.Issuers = []certmagic.Issuer{
		certmagic.NewACMEIssuer(magic, certmagic.ACMEIssuer{
			CA:                      certmagic.LetsEncryptProductionCA,
			Email:                   email,
			Agreed:                  true,
			DNS01Solver:             &certmagic.DNS01Solver{DNSManager: certmagic.DNSManager{DNSProvider: dnsProvider}},
			DisableHTTPChallenge:    true,
			DisableTLSALPNChallenge: true,
		}),
	}
	if staging {
		magic.Issuers = []certmagic.Issuer{
			certmagic.NewACMEIssuer(magic, certmagic.ACMEIssuer{
				CA:                      certmagic.LetsEncryptStagingCA,
				Email:                   email,
				Agreed:                  true,
				DNS01Solver:             &certmagic.DNS01Solver{DNSManager: certmagic.DNSManager{DNSProvider: dnsProvider}},
				DisableHTTPChallenge:    true,
				DisableTLSALPNChallenge: true,
			}),
		}
	}

	// Obtain certificates for the wildcard and the apex domain
	domains := []string{"*." + domain}
	if err := magic.ManageSync(context.Background(), domains); err != nil {
		return nil, fmt.Errorf("obtaining TLS cert for %v: %w", domains, err)
	}

	return magic.TLSConfig(), nil
}
