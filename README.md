# devkit

Local-first, Docker-based dev environments for self-hosted tools.

## Tools

| Tool | Description | Status |
|------|-------------|--------|
| [trigger-dev](./trigger-dev/) | Trigger.dev v4 — background jobs, workflows, human-in-the-loop | ✅ |

## Usage

Each tool follows the same pattern:

```bash
cd <tool>/
make setup   # one-time: pulls infra + installs deps
make up      # start everything
make down    # stop everything
```

See each tool's README for details.
