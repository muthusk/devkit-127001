# devkit Reverse Proxy

A lightweight Go reverse proxy with TLS that provides subdomain-based routing and a dashboard for all devkit tools.

## Prerequisites

- Go 1.21+ (`brew install go`)
- OpenSSL (pre-installed on macOS)
- dnsmasq — resolves `*.dev.kit` to `127.0.0.1`

## Local Setup (one-time)

### 1. DNS — resolve `*.dev.kit` to localhost

```bash
brew install dnsmasq
echo 'address=/.dev.kit/127.0.0.1' >> $(brew --prefix)/etc/dnsmasq.conf
sudo brew services restart dnsmasq

# Tell macOS to use dnsmasq for .dev.kit
sudo mkdir -p /etc/resolver
echo 'nameserver 127.0.0.1' | sudo tee /etc/resolver/dev.kit
```

Verify: `dig home.dev.kit @127.0.0.1` should return `127.0.0.1`.

### 2. TLS — generate wildcard cert

```bash
cd proxy/
./scripts/setup-devkit-ssl.sh   # creates local CA + wildcard cert, trusts CA in keychain
```

This creates a root CA in `~/.devkit-ssl/` and a wildcard server cert in `proxy/cert/`. The CA is installed into the macOS system keychain so browsers trust it automatically. Run once — certs are valid for 825 days.

The script also creates a combined CA bundle (`~/.devkit-ssl/combined-ca.pem`) and offers to add these exports to `~/.zshrc`:

```bash
export NODE_EXTRA_CA_CERTS="$HOME/.devkit-ssl/ca.pem"
export SSL_CERT_FILE="$HOME/.devkit-ssl/combined-ca.pem"
```

These ensure Node.js and OpenSSL-based tools (homebrew curl, python, etc.) trust the devkit CA.

### 3. Port forwarding — HTTPS on port 443 without root

The proxy listens on port 7443 (unprivileged). To access `https://home.dev.kit` without specifying a port, forward 443 → 7443 using socat + launchd:

```bash
brew install socat
sudo ./scripts/setup-socat.sh       # installs LaunchDaemon (one-time, survives reboots)
```

This creates a LaunchDaemon that runs `socat TCP-LISTEN:443,fork,reuseaddr TCP:127.0.0.1:7443` as root. It starts automatically on boot and restarts if it crashes.

To remove:

```bash
sudo ./scripts/setup-socat.sh --remove
```

Logs: `/tmp/dev.kit.port-forward.log`

## Quick Start

```bash
make up       # build and start the proxy with TLS on :7443
make down     # stop the proxy
make status   # check if running and show registered tools
```

Dashboard: https://home.dev.kit

## How It Works

The proxy terminates TLS using a wildcard cert for `*.dev.kit` and routes requests by subdomain:

- `https://home.dev.kit` → dashboard + admin API
- `https://temporal.dev.kit` → forwards to Temporal UI on its local port
- `https://keycloak.dev.kit` → forwards to Keycloak on its local port
- `https://trigger-dev.dev.kit` → forwards to Trigger.dev on its local port

Each tool registers its name and port. The proxy creates a subdomain route automatically.

## API

### Register a tool

```
POST https://home.dev.kit/proxy/register
Content-Type: application/json

{
  "name": "myapp",
  "port": 8080
}
```

The tool becomes available at `https://myapp.dev.kit`.

### Deregister a tool

```
POST https://home.dev.kit/proxy/deregister
Content-Type: application/json

{
  "name": "myapp"
}
```

## Integration

Tools call these endpoints from their Makefiles as a best-effort step after `make up` / before `make down`. The proxy is never a hard dependency — tools work standalone without it.

## Configuration

All config is in `.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| `TLS_PORT` | `7443` | Port the proxy listens on (TLS) |
| `TLS_CERT` | `cert/_wildcard.dev.kit-cert.pem` | Path to wildcard TLS certificate |
| `TLS_KEY` | `cert/_wildcard.dev.kit-key.pem` | Path to TLS private key |
| `HEALTH_CHECK_INTERVAL` | `5m` | How often to check tool health |
| `STATE_FILE` | `state.json` | Where to persist route state for crash recovery |
