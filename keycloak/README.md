# Keycloak Local Dev Environment

Self-contained Keycloak identity provider with PostgreSQL backend, running in Docker.

## Quick Start

```bash
make up       # Start Keycloak + Postgres
make seed     # Create sample realm, client, and users
make down     # Graceful shutdown (preserves data)
```

## Access

| Service         | URL                              | Credentials   |
|-----------------|----------------------------------|---------------|
| Admin Console   | http://localhost:8180/keycloak    | admin / admin |
| OIDC Discovery  | http://localhost:8180/keycloak/realms/devkit-sample/.well-known/openid-configuration | — |

## Seed Data

`make seed` creates the following via the Keycloak Admin REST API:

- **Realm:** `devkit-sample`
- **Client:** `devkit-app` (public OIDC client, standard flow + direct access grants)
  - Redirect URIs: `http://localhost:*`
- **Users:**
  - `alice` / `alice123`
  - `bob` / `bob123`

Seeding is idempotent — safe to run multiple times.

## Makefile Targets

Run `make help` for the full list.

| Target     | Description                                         |
|------------|-----------------------------------------------------|
| `up`       | Start Keycloak and Postgres                         |
| `down`     | Graceful shutdown (preserves volumes)               |
| `restart`  | Full restart (down + up)                            |
| `status`   | Show container health                               |
| `logs`     | Tail all logs                                       |
| `seed`     | Create sample realm, client, and users              |
| `shell`    | Interactive bash in Keycloak container              |
| `clean`    | Remove containers and volumes (optionally images)   |

## Proxy Integration

If the devkit proxy is running at `localhost:7001`, `make up` automatically registers Keycloak at `/keycloak`. Access via `http://localhost:7001/keycloak`.

If the proxy isn't running, the registration is skipped and the direct URL is printed.

## Configuration

All settings are in `.env`. Ports are auto-detected at startup via `shared/scripts/find-port.sh` if available, falling back to `.env` defaults.
