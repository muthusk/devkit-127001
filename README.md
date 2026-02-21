# devkit

Local-first, Docker-based dev environments for self-hosted tools.

Each tool is a self-contained directory with its own `docker-compose.yml`, `Makefile`, and documentation. Clone the repo, pick a tool, and `make up`.

## Tools

| Tool | Description | Status |
|------|-------------|--------|
| [trigger-dev](./trigger-dev/) | Trigger.dev v4 — background jobs, workflows, human-in-the-loop | ✅ |
| [temporal](./temporal/) | Temporal — durable workflow orchestration with Go + TypeScript workers | ✅ |
| keycloak | Keycloak — identity and access management, OIDC/SAML provider | 🔜 |
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