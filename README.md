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

A shared Go-based reverse proxy (`proxy/`) runs on port 7001 and provides path-based routing to all tools. Each tool registers itself on `make up` and deregisters on `make down`. The proxy is optional — tools work standalone via their direct ports.

Dashboard: `http://localhost:7001/`

## Planned: Local DNS + TLS (`.devkit` domains)

We're migrating from path-based routing (`localhost:7001/temporal`) to subdomain-based routing with HTTPS:

| Current | Planned |
|---------|---------|
| `http://localhost:8080` | `https://temporal.devkit` |
| `http://localhost:8030` | `https://trigger.devkit` |
| `http://localhost:8180` | `https://keycloak.devkit` |

### How it will work

1. **Local DNS** — `dnsmasq` resolves `*.devkit` to `127.0.0.1` via `/etc/resolver/devkit`
2. **Local CA + wildcard cert** — `mkcert` creates a trusted CA and `*.devkit` certificate. Browsers (Chrome, Firefox, Safari) and `curl` trust it automatically.
3. **TLS termination** — the Go proxy serves HTTPS using the wildcard cert (`ListenAndServeTLS`), routing by `Host` header instead of URL path
4. **Port 443 without root** — `pfctl` forwards port 443 to the proxy's unprivileged port (e.g., 7443), so the proxy runs without sudo

This eliminates the need for path-prefix configuration in apps, which causes issues with asset paths and redirects for tools that don't support a base-path setting.