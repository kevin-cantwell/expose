# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`expose` is a personal ngrok replacement that tunnels local HTTP ports to `<subdomain>.<EXPOSE_DOMAIN>` via secure HTTPS. The server runs on a Linux host you control. Auth uses GitHub OAuth device flow — only the configured `EXPOSE_ALLOWED_USER` GitHub username can connect.

## Commands

```bash
# Build
go build -o expose .

# Build for deployment (Linux amd64)
make build-linux

# Deploy to server (build + scp + systemctl restart)
# Requires local.mk — see local.mk.example
make deploy

# Watch server logs
make logs

# Run locally (requires EXPOSE_SERVER and EXPOSE_GITHUB_CLIENT_ID env vars)
expose login              # first-time GitHub auth
expose :3000              # tunnel localhost:3000 with auto subdomain
expose :3000 -s myapp     # tunnel as myapp.<EXPOSE_DOMAIN>
expose ls                 # list active tunnels on this machine

# Server (on droplet, requires /etc/expose.env — see deploy/expose.env.example)
expose serve
```

## Configuration

**Client env vars** (set in shell or `.envrc`):
- `EXPOSE_SERVER` — your domain (e.g. `example.com`) **required**
- `EXPOSE_GITHUB_CLIENT_ID` — GitHub OAuth App client ID **required**

**Server env vars** (set in `/etc/expose.env` on the server):
- `EXPOSE_DOMAIN` — base domain **required**
- `EXPOSE_ALLOWED_USER` — GitHub username allowed to connect **required**
- `DO_AUTH_TOKEN` — DigitalOcean API token for DNS-01 cert challenge **required**
- `EXPOSE_ACME_EMAIL` — email for Let's Encrypt
- `EXPOSE_CERT_DIR` — cert storage path (default `/var/lib/expose/certs`)

**Deploy vars** (set in `local.mk`, gitignored):
- `DROPLET_IP`, `DROPLET_USER`, `SSH_KEY` — see `local.mk.example`

## Architecture

Single Go binary with two subcommands: `serve` and a default tunnel command, plus `login` and `ls`. Uses Kong for CLI parsing.

**Server (`expose serve`)** — runs on your Linux host, owns ports 443/80:
- `server/certs.go`: certmagic + libdns/digitalocean for automatic Let's Encrypt wildcard TLS (`*.<domain>`) via DNS-01 challenge.
- `server/server.go`: Routes by `Host` header. `expose.<domain>/connect` is the WebSocket endpoint for tunnel clients. Other subdomains are proxied through their registered tunnel.
- `server/auth.go`: Validates GitHub OAuth tokens via `GET /api.github.com/user`, checks `login == EXPOSE_ALLOWED_USER`.
- `server/tunnel.go`: Per-tunnel state — holds a `*yamux.Session`. Opens a yamux stream per HTTP request, writes the raw request, reads the response.

**Client (`expose :3000`)** — runs locally:
- `client/client.go`: Connects to `wss://expose.<domain>/connect` with the GitHub token. Wraps the WebSocket as a yamux server, accepts streams, dials `localhost:<port>`, pipes request/response back.
- `client/tui.go`: Prints connection info and a live request log to stdout.
- `client/state.go`: Writes `~/.config/expose/tunnels/<subdomain>.json` while connected so `expose ls` can find it.

**Tunnel transport**: WebSocket (coder/websocket) + yamux multiplexing. The HTTP server uses `yamux.Client` (opens streams per request); the local client uses `yamux.Server` (accepts streams). Each HTTP request is one yamux stream carrying raw `req.Write` / `http.ReadResponse` bytes.

**Subdomain generation**: `internal/words.go` — adjective + noun word lists, random two-word combo like `brave-llama`.

## Infrastructure Setup (one-time)

```bash
# 1. Add wildcard DNS A record pointing to your server
doctl compute domain records create <domain> \
  --record-type A --record-name '*' --record-data <server-ip>

# 2. Create a GitHub OAuth App at github.com/settings/applications/new
#    Enable Device Flow. Callback URL can be anything valid.
#    Set EXPOSE_GITHUB_CLIENT_ID to the client ID.

# 3. On the server: create /etc/expose.env (see deploy/expose.env.example)
mkdir -p /var/lib/expose/certs

# 4. Install and start the service
scp deploy/expose.service root@<server>:/etc/systemd/system/expose.service
ssh root@<server> "systemctl daemon-reload && systemctl enable expose"
make deploy
```
