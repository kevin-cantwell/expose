# expose

A self-hosted ngrok replacement. Tunnels local HTTP ports to `<subdomain>.<your-domain>` with automatic HTTPS via Let's Encrypt.

```
expose :3000
```
```
expose connected

  URL:    https://brave-llama.example.com
  Local:  http://localhost:3000
  Time:   2026-03-25 10:00:00

  METHOD  PATH                                      STATUS  LATENCY
  ─────────────────────────────────────────────────────────────────
  GET     /api/users                                200     12ms
  POST    /api/users                                201     45ms
```

## How it works

A server process runs on a Linux host you control and owns ports 80/443. When you run `expose :3000` locally, it opens a WebSocket connection to the server and registers a subdomain. Incoming HTTPS requests are multiplexed over that connection ([yamux](https://github.com/hashicorp/yamux)) and forwarded to your local port. TLS is handled automatically via Let's Encrypt wildcard certs (DNS-01 challenge).

Auth uses [GitHub OAuth Device Flow](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#device-flow) — no passwords, no tokens to manage. You configure a single allowed GitHub username server-side.

## Prerequisites

- A Linux server with ports 80 and 443 open
- A domain with DNS managed by DigitalOcean
- A [GitHub OAuth App](https://github.com/settings/applications/new) with Device Flow enabled
- A DigitalOcean API token with DNS write access
- Go 1.22+ (for building)

> **DNS provider**: The server uses [certmagic](https://github.com/caddyserver/certmagic) with [libdns/digitalocean](https://github.com/libdns/digitalocean) for the DNS-01 challenge. If your DNS isn't on DigitalOcean, swap in the [libdns provider](https://github.com/libdns) for your registrar and update `server/certs.go`.

## Server setup

**1. Add a wildcard DNS A record** pointing to your server:

```bash
doctl compute domain records create example.com \
  --record-type A --record-name '*' --record-data <server-ip>
```

**2. Create a GitHub OAuth App** at [github.com/settings/applications/new](https://github.com/settings/applications/new):
- Enable **Device Flow**
- Homepage URL: `https://expose.example.com`
- Callback URL: anything valid (not used by device flow)
- Note the **Client ID** — you don't need a client secret

**3. Create `/etc/expose.env`** on your server (see `deploy/expose.env.example`):

```bash
EXPOSE_DOMAIN=example.com
EXPOSE_ALLOWED_USER=your-github-username
DO_AUTH_TOKEN=dop_v1_...
EXPOSE_ACME_EMAIL=you@example.com
EXPOSE_CERT_DIR=/var/lib/expose/certs
```

```bash
chmod 600 /etc/expose.env
mkdir -p /var/lib/expose/certs
```

**4. Install the systemd service**:

```bash
# Build for Linux
make build-linux

# Copy binary and service file
scp expose-linux root@<server>:/usr/local/bin/expose
scp deploy/expose.service root@<server>:/etc/systemd/system/expose.service

# Enable and start
ssh root@<server> "systemctl daemon-reload && systemctl enable --now expose"
```

Or use `make deploy` after configuring `local.mk` (see below).

## Client setup

**1. Clone and build**:

```bash
git clone https://github.com/kevin-cantwell/expose
cd expose
go build -o expose .
cp expose ~/.local/bin/  # or anywhere on your PATH
```

**2. Set environment variables** (e.g. in `~/.zshrc` or a `.envrc`):

```bash
export EXPOSE_SERVER=example.com
export EXPOSE_GITHUB_CLIENT_ID=<your-oauth-app-client-id>
```

**3. Log in**:

```bash
expose login
# Opens github.com/login/device — enter the displayed code
```

**4. Tunnel something**:

```bash
expose :3000                  # auto-generated subdomain
expose :3000 -s myapp         # → myapp.example.com
expose ls                     # list active tunnels
```

## Deployment automation

Copy `local.mk.example` to `local.mk` (gitignored) and fill in your values:

```makefile
DROPLET_IP   := <server-ip>
DROPLET_USER := root
SSH_KEY      := ~/.ssh/your_key
```

Then:

```bash
make deploy    # build for linux, scp, restart service
make logs      # tail journald logs
make restart   # restart the service
```

## Configuration reference

| Env var | Side | Description |
|---|---|---|
| `EXPOSE_SERVER` | client | Your domain (e.g. `example.com`) |
| `EXPOSE_GITHUB_CLIENT_ID` | client | GitHub OAuth App client ID |
| `EXPOSE_DOMAIN` | server | Base domain for tunnels |
| `EXPOSE_ALLOWED_USER` | server | GitHub username allowed to connect |
| `DO_AUTH_TOKEN` | server | DigitalOcean API token (DNS-01) |
| `EXPOSE_ACME_EMAIL` | server | Email for Let's Encrypt |
| `EXPOSE_CERT_DIR` | server | TLS cert storage path |
| `EXPOSE_STAGING` | server | Use Let's Encrypt staging (`true`/`false`) |
