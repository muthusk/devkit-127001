# devkit

Local-first, Docker-based dev environments for self-hosted tools.

Each tool is a self-contained directory with its own `docker-compose.yml`, `Makefile`, and documentation. Clone the repo, pick a tool, and `make up`.

## Tools

| Tool | Description | Status |
|------|-------------|--------|
| [trigger-dev](./trigger-dev/) | Trigger.dev v4 — background jobs, workflows, human-in-the-loop | ✅ |
| [temporal](./temporal/) | Temporal — durable workflow orchestration with Go + TypeScript workers | ✅ |
| [keycloak](./keycloak/) | Keycloak — identity and access management, OIDC/SAML provider | ✅ |
| slurm | Slurm — HPC workload manager and job scheduler | 🔜 |

## Usage

Each tool follows the same pattern:
```bash
cd <tool>/
make up      # start everything
make down    # stop everything
make seed    # load example data / workflows (where applicable)
make help    # list all available commands
```

See each tool's README for details.

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (includes Compose v2)
- Make (`xcode-select --install` on macOS)

No other host dependencies required — language runtimes, databases, and tooling all run inside containers.

## Reverse Proxy

A shared Go-based reverse proxy (`proxy/`) provides TLS-terminated subdomain routing for all tools. Each tool registers itself on `make up` and deregisters on `make down`. The proxy is optional — tools work standalone via their direct ports.

Dashboard: `https://home.dev.kit`

### How it works

1. **Local DNS** — `dnsmasq` resolves `*.dev.kit` to `127.0.0.1` via `/etc/resolver/dev.kit`
2. **Local CA + wildcard cert** — `setup-devkit-ssl.sh` uses OpenSSL to create a local CA and `*.dev.kit` certificate, then trusts the CA in the macOS keychain.
3. **TLS termination** — the Go proxy serves HTTPS using the wildcard cert, routing by `Host` header
4. **Port 443 without root** — `socat` forwards port 443 to the proxy's unprivileged port (7443)

| Tool | URL |
|------|-----|
| Dashboard | `https://home.dev.kit` |
| Temporal | `https://temporal.dev.kit` |
| Trigger.dev | `https://trigger-dev.dev.kit` |
| Keycloak | `https://keycloak.dev.kit` |
