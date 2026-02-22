# devkit Reverse Proxy

A lightweight Go reverse proxy that provides a single entry point and dashboard for all devkit tools.

## Prerequisites

- Go 1.21+ (`brew install go`)

## Quick Start

```bash
make up       # build and start the proxy on localhost:7001
make down     # stop the proxy
make status   # check if running and show registered tools
```

Dashboard: http://localhost:7001

## How It Works

The proxy listens on `localhost:7001` and routes requests by path prefix to registered tools. It also runs periodic health checks and displays status on a homepage dashboard.

## API

### Register a tool

```
POST /proxy/register
Content-Type: application/json

{
  "name": "keycloak",
  "port": 8180,
  "healthURL": "http://localhost:8180/health/ready",
  "path": "/keycloak"
}
```

### Deregister a tool

```
POST /proxy/deregister
Content-Type: application/json

{
  "name": "keycloak"
}
```

## Integration

Tools call these endpoints from their Makefiles as a best-effort step after `make up` / before `make down`. The proxy is never a hard dependency — tools work standalone without it.

## Configuration

All config is in `.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| `PROXY_PORT` | `7001` | Port the proxy listens on |
| `HEALTH_CHECK_INTERVAL` | `5m` | How often to check tool health |
| `STATE_FILE` | `state.json` | Where to persist route state for crash recovery |
