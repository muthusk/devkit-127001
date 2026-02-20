# Trigger.dev — Local Dev

Self-hosted Trigger.dev v4 with Docker + preconfigured example tasks.

## Prerequisites

- Docker Desktop for macOS (20.10.0+) with 8GB+ RAM
- Docker Compose (2.20.0+)
- Node.js 20+ and npm

## Quick Start

```bash
# 1. One-time setup (clones infra, installs deps)
make setup

# 2. Start everything
make up

# 3. Get your magic login link (wait ~30-60s after startup)
make logs-magic-link

# 4. Open http://localhost:8030, sign in via magic link
#    Create an organization and project in the UI

# 5. Link task project (pass project ref from dashboard)
make init PROJECT=proj_xxxxx

# 6. Login to local Docker registry (password: very-secure-indeed)
make registry-login

# 7. Start dev watcher
make dev

# 8. (In another terminal) Deploy tasks
make deploy
```

## Commands

| Command                | Description                                         |
|------------------------|-----------------------------------------------------|
| `make setup`           | One-time: clone infra + install deps                |
| `make up`              | Start all infrastructure (checks for updates)       |
| `make down`            | Stop all infrastructure                             |
| `make restart`         | Restart all infrastructure                          |
| `make logs`            | Tail all container logs                             |
| `make logs-magic-link` | Watch for magic login links                         |
| `make status`          | Show container status                               |
| `make init PROJECT=x`  | Initialize task project against local instance      |
| `make dev`             | Run trigger dev watcher                             |
| `make deploy`          | Deploy tasks to local instance                      |
| `make registry-login`  | Login to the local Docker registry                  |
| `make upgrade`         | Upgrade to latest Trigger.dev release               |
| `make clean`           | Stop infra, remove volumes (⚠️ destroys data)       |
| `make nuke`            | Full wipe: containers, volumes, images, caches (☢️)  |
| `make health`          | Check webapp health endpoint                        |
| `make worker-token`    | Show the worker token from logs                     |

## Update Checks

`make up` checks GitHub for newer releases (cached for 4h). Minor/patch
updates offer inline upgrade. Major version jumps warn only.

```
$ make up
  ⬆️  Trigger.dev v4.3.5 is available (you're on v4.3.3)
  Upgrade now? [y/N] y
  📦 Upgrading v4.3.3 → v4.3.5...
  ✅ Upgraded to v4.3.5

  🗑️  Old image v4.3.3 (~1.2GB) is not in use.
  Delete it to free space? [y/N] y
  ✅ Deleted old image (1.2GB freed).

🚀 Starting infrastructure on v4.3.5...
```

To skip:
```bash
SKIP_UPDATE_CHECK=1 make up                     # once
echo "UPDATE_CHECK=false" >> .trigger-local      # permanently (gitignored)
```

## Example Tasks

| File                            | Type                        | Description                              |
|---------------------------------|-----------------------------|------------------------------------------|
| `src/trigger/dummy-wait.ts`     | Wait / sleep                | Just waits — test infra + trace view     |
| `src/trigger/event-processor.ts`| Event-driven background job | Routes events with retries + queues      |
| `src/trigger/approval-flow.ts`  | Human-in-the-loop           | Pauses for manual approval via waitpoint |

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Docker Compose (infra/)                        │
│  ┌──────────┐ ┌───────┐ ┌───────┐ ┌──────────┐ │
│  │ Webapp   │ │Postgre│ │ Redis │ │Supervisor│ │
│  │ :8030    │ │ :5432  │ │ :6379 │ │          │ │
│  └──────────┘ └───────┘ └───────┘ └──────────┘ │
│  ┌──────────┐ ┌──────────────┐ ┌─────────────┐ │
│  │ Registry │ │ Socket Proxy │ │   MinIO     │ │
│  │ :5000    │ │              │ │ :9000/:9001 │ │
│  └──────────┘ └──────────────┘ └─────────────┘ │
└─────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────┐
│  Task Project (this directory)                  │
│  src/trigger/*.ts  →  `make dev` / `make deploy`│
└─────────────────────────────────────────────────┘
```

## Default Credentials (local dev only!)

| Service   | URL                    | User           | Password             |
|-----------|------------------------|----------------|----------------------|
| Webapp    | http://localhost:8030   | magic link     | (check logs)         |
| Registry  | localhost:5000         | registry-user  | very-secure-indeed   |
| MinIO     | http://localhost:9001   | admin          | very-safe-password   |

## Team Onboarding

```bash
git clone <this-repo>
cd devkit/trigger-dev
make setup
make up
```
