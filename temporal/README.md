# Temporal Local Dev Environment

A fully self-contained, repeatable Temporal development environment with **Go** and **TypeScript** workers running side-by-side. Designed for small teams (2-5 developers) who want to learn and build with Temporal locally.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        Docker Network                            │
│                                                                  │
│  ┌──────────┐    ┌──────────────────┐    ┌──────────────────┐   │
│  │ Postgres │◄───│  Temporal Server  │───►│   Temporal UI    │   │
│  │  :5432   │    │  (auto-setup)    │    │   :8080          │   │
│  └──────────┘    │  :7233 (gRPC)    │    └──────────────────┘   │
│                  └────────┬─────────┘                            │
│                           │                                      │
│              ┌────────────┴────────────┐                         │
│              │                         │                         │
│  ┌───────────▼──────────┐  ┌──────────▼───────────┐            │
│  │   Go Worker          │  │   TS Worker           │            │
│  │   go-task-queue      │  │   ts-task-queue       │            │
│  │                      │  │                       │            │
│  │ • EventProcessing    │  │ • eventProcessing     │            │
│  │ • DummyWait          │  │ • dummyWait           │            │
│  │ • HumanApproval      │  │ • humanApproval       │            │
│  │ • Saga               │  │ • saga                │            │
│  │ • ChildFanout        │  │ • childFanout         │            │
│  └──────────────────────┘  └───────────────────────┘            │
│                                                                  │
│  ┌──────────────────────┐                                       │
│  │  Admin Tools (CLI)   │                                       │
│  └──────────────────────┘                                       │
└──────────────────────────────────────────────────────────────────┘
```

Both workers implement the **same 5 workflow patterns** with slight variations per language. They run on **separate task queues** so both can operate simultaneously without conflicts.

---

## Prerequisites

| Tool    | Version  | Check                  |
|---------|----------|------------------------|
| Docker  | 20.10+   | `docker --version`     |
| Compose | v2+      | `docker compose version` |
| Make    | Any      | `make --version`       |

> **macOS:** Docker Desktop includes both Docker and Compose. Make is available via Xcode Command Line Tools (`xcode-select --install`).

---

## Quick Start

```bash
# 1. Clone the repo and enter it
git clone <your-repo-url> temporal
cd temporal

# 2. Start everything
make up

# 3. Trigger example workflows
make seed

# 4. Open the Temporal UI
open http://localhost:8080
```

That's it. All 10 workflows (5 Go + 5 TS) will be visible and running in the UI.

---

## What's Inside — The 5 Workflow Patterns

Each pattern is implemented in both Go and TypeScript. The TS versions have deliberate variations to teach additional concepts.

### 1. Event-Driven Background Job

**Concept:** Sequential activity pipeline with retry policies.

| | Go Version | TS Version |
|---|---|---|
| **Activities** | validate → process → notify (sequential) | validate + enrich (parallel) → process → notify |
| **Key lesson** | Activity chaining, error propagation | `Promise.all` for concurrent activities |

```bash
make seed-event-go    # Trigger Go version
make seed-event-ts    # Trigger TS version
```

### 2. Dummy Wait

**Concept:** Timer-based workflow that does nothing but sleep.

| | Go Version | TS Version |
|---|---|---|
| **Behavior** | Single `workflow.Sleep()` | Multi-stage: 3 sleeps with checkpoint logging |
| **Key lesson** | Timers, cancellation, UI visibility | State tracking across timer stages |

```bash
make seed-wait-go     # Trigger Go version (30s)
make seed-wait-ts     # Trigger TS version (30s, 3 stages)
```

### 3. Human Approval (Signal-based)

**Concept:** Workflow blocks waiting for external human input.

| | Go Version | TS Version |
|---|---|---|
| **Signal** | `approval` channel with timeout | Same + **Query handler** for status checks |
| **Key lesson** | Signals, selectors, timeouts | `defineQuery`, `setHandler`, `condition` |

```bash
make seed-approval-go                       # Start Go approval workflow
make seed-approval-ts                       # Start TS approval workflow
make signal-approve WID=approval-go-xxx     # Approve it
make signal-reject WID=approval-ts-xxx      # Reject it
make query-approval-ts WID=approval-ts-xxx  # Check status (TS only)
```

### 4. Saga / Compensation

**Concept:** Multi-step transaction with automatic rollback on failure.

| | Go Version | TS Version |
|---|---|---|
| **Steps** | 3 steps: reserve → charge → ship | 4 steps: reserve → charge → ship → confirm |
| **Failure** | Ship has ~30% failure rate | Confirm has ~30% failure rate |
| **Compensation twist** | Standard reverse compensations | `cancelShipping` itself can fail (~20%) |
| **Key lesson** | Saga pattern, compensation ordering | Nested error handling in compensations |

```bash
make seed-saga-go     # Trigger Go version
make seed-saga-ts     # Trigger TS version
```

> **Tip:** Run the saga multiple times to see both success and compensation paths.

### 5. Child Workflow Fanout

**Concept:** Parent spawns N child workflows and aggregates results.

| | Go Version | TS Version |
|---|---|---|
| **Parallelism** | All children launched at once | Batched with configurable concurrency limit (default: 3) |
| **Key lesson** | Child workflows, `Future` collection | Controlled parallelism, batch processing |

```bash
make seed-child-go    # Trigger Go version (5 items, all parallel)
make seed-child-ts    # Trigger TS version (5 items, 3 at a time)
```

---

## Makefile Reference

### Environment

| Command | Description |
|---------|-------------|
| `make up` | Start all services, build workers, wait for ready |
| `make down` | Stop everything and delete volumes (clean slate) |
| `make restart` | Full `down` + `up` cycle |
| `make stop` | Stop without removing volumes (preserves data) |
| `make start` | Resume stopped services |
| `make status` | Show container status and Temporal health |

### Logs

| Command | Description |
|---------|-------------|
| `make logs` | Tail all container logs |
| `make logs-go` | Tail Go worker only |
| `make logs-ts` | Tail TS worker only |
| `make logs-server` | Tail Temporal server only |

### Seeding

| Command | Description |
|---------|-------------|
| `make seed` | Trigger all 10 workflows |
| `make seed-go` | Trigger all 5 Go workflows |
| `make seed-ts` | Trigger all 5 TS workflows |
| `make seed-event-go` | Individual Go event workflow |
| `make seed-saga-ts` | Individual TS saga workflow |
| *(etc.)* | See `make help` for full list |

### Interactions

| Command | Description |
|---------|-------------|
| `make signal-approve WID=<id>` | Send approval signal |
| `make signal-reject WID=<id>` | Send rejection signal |
| `make query-approval-ts WID=<id>` | Query TS approval status |

### Development

| Command | Description |
|---------|-------------|
| `make rebuild-go` | Rebuild + restart Go worker only |
| `make rebuild-ts` | Rebuild + restart TS worker only |
| `make shell` | Shell into admin-tools container |
| `make clean` | Full cleanup including built images |
| `make help` | Show all available targets |

---

## Port Mapping

| Service | Host Port | Purpose |
|---------|-----------|---------|
| Temporal gRPC | `7233` | Worker and CLI connections |
| Temporal HTTP | `7243` | HTTP for REST connections |
| Temporal UI | `8080` | Web dashboard |
| PostgreSQL | `5432` | Direct DB access (optional) |

All ports are configurable via `.env`.

---

## Environment Variables

See `.env` for all configuration. Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `TEMPORAL_VERSION` | `latest` | Temporal server image tag |
| `TEMPORAL_UI_PORT` | `8080` | Host port for the web UI |
| `TEMPORAL_GRPC_PORT` | `7233` | Host port for gRPC |
| `TEMPORAL_HTTP_PORT` | `7243` | Host port for gRPC |
| `GO_TASK_QUEUE` | `go-task-queue` | Task queue for Go worker |
| `TS_TASK_QUEUE` | `ts-task-queue` | Task queue for TS worker |
| `POSTGRES_PASSWORD` | `temporal` | DB password (dev only!) |

---

## How Repeatability Works

This setup is designed so **any team member can go from zero to running in one command**:

1. **`temporalio/auto-setup`** handles all database schema migrations automatically on first boot. If the schema already exists, it's a no-op.

2. **Named Docker volumes** (`temporal-postgres-data`) persist data across `stop`/`start` cycles. Use `make down` (with `-v`) for a true clean slate.

3. **Health checks + `depends_on`** ensure services start in the correct order. The Go and TS workers won't attempt to connect until the Temporal server is healthy.

4. **`scripts/wait-for-temporal.sh`** provides an additional readiness gate before `make up` reports success.

5. **`.env` file** pins all image versions so every team member gets identical environments.

6. **No host dependencies beyond Docker and Make.** Go and Node.js are only needed inside the containers.

---

## How to Add Your Own Workflow

### Go

1. Create `worker-go/workflows/my_workflow.go` with your workflow function
2. Create `worker-go/activities/my_activities.go` with any activities
3. Register both in `worker-go/main.go`
4. `make rebuild-go`
5. Start it: `make shell` → `temporal workflow start --task-queue go-task-queue --type MyWorkflow --input '{}'`

### TypeScript

1. Create `worker-ts/src/workflows/myWorkflow.ts`
2. Export it from `worker-ts/src/workflows/index.ts`
3. Create `worker-ts/src/activities/myActivities.ts`
4. Import and spread activities in `worker-ts/src/worker.ts`
5. `make rebuild-ts`

See the worker-specific READMEs for detailed patterns:
- [`worker-go/README.md`](worker-go/README.md)
- [`worker-ts/README.md`](worker-ts/README.md)

---

## Adding a Third Worker (e.g., Python)

The architecture supports adding workers in any Temporal-supported language:

1. Create `worker-python/` with a `Dockerfile`
2. Add a new service in `docker-compose.yml` with a unique task queue (e.g., `py-task-queue`)
3. Add corresponding `seed-python.sh` and Makefile targets
4. `make up` — it joins the same Temporal server automatically

---

## Troubleshooting

### Temporal server won't start

```bash
# Check Postgres is healthy
docker compose logs postgres

# Check server logs for migration errors
docker compose logs temporal-server

# Nuclear option: clean slate
make clean && make up
```

### Worker can't connect

```bash
# Verify server is healthy
make status

# Check worker logs
make logs-go   # or make logs-ts

# Common cause: server not ready yet. Workers auto-retry on failure.
```

### Port conflicts

If port `8080` or `7233` or `7243` is already in use, change it in `.env`:
```
TEMPORAL_UI_PORT=8081
TEMPORAL_GRPC_PORT=7234
```
Then `make restart`.

### Stale Docker images

```bash
make clean    # Removes images + volumes
make up       # Fresh build
```

### "No workers running" in the UI

This means the worker container crashed or can't connect. Check:
```bash
make logs-go   # Check for panic/error
make logs-ts   # Check for TypeScript compilation errors
```

---

## Team Onboarding Checklist

- [ ] Install Docker Desktop
- [ ] Clone this repo
- [ ] Run `make up` — verify all 6 containers are running
- [ ] Run `make seed` — verify workflows appear in UI at `localhost:8080`
- [ ] Run `make signal-approve WID=approval-go-<timestamp>` — verify it completes
- [ ] Read `worker-go/README.md` or `worker-ts/README.md` depending on your stack
- [ ] Try modifying an activity, then `make rebuild-go` or `make rebuild-ts`
- [ ] Open `make shell` and explore `temporal workflow list`, `temporal workflow show`

---

## Resources

- [Temporal Documentation](https://docs.temporal.io/)
- [Temporal Go SDK](https://pkg.go.dev/go.temporal.io/sdk)
- [Temporal TypeScript SDK](https://typescript.temporal.io/)
- [Temporal CLI Reference](https://docs.temporal.io/cli)
- [Temporal UI Guide](https://docs.temporal.io/web-ui)
