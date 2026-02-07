# Iceberg

Iceberg is a self-hosting control plane with:
- `iceberg` CLI for provisioning and operations
- `iceberg-agent` for on-server deployment/health APIs
- Built-in service catalog (Uptime Kuma, Plausible, n8n, Gitea, Supabase)

## Architecture

This repository follows a hexagonal structure:
- `internal/domain`: domain entities (`Server`, `Service`)
- `internal/ports`: repository contracts
- `internal/app`: use-cases (`ServerUseCase`, `ServiceUseCase`)
- `internal/adapters`: external adapters (HTTP clients, Postgres/sqlc, migration runner)
- `internal/cli`, `internal/agent`: delivery adapters (CLI + HTTP API)

## Database management (`sqlc` + `atlas` + `golang-migrate`)

- SQL schema: `db/schema/schema.sql`
- SQL migrations: `db/migrations/*.sql`
- SQL queries: `db/query/*.sql`
- sqlc config: `sqlc.yaml`
- Atlas config: `atlas.hcl`
- Generated sqlc code: `internal/adapters/db/sqlc`

### Commands

```bash
make sqlc
DATABASE_URL=postgres://... make db-migrate-up
DATABASE_URL=postgres://... make db-migrate-down
```

## Build and test

```bash
make build
make test
make lint
```

## CLI commands

```bash
iceberg init
iceberg server create
iceberg server list
iceberg server destroy <name>
iceberg deploy uptime-kuma --domain status.example.com
iceberg deploy --image nginx:latest --name my-nginx --port 80
iceberg status
iceberg logs <service>
iceberg destroy <service>
iceberg catalog
```

`iceberg server create` now bootstraps the server automatically:
- Builds a Linux `iceberg-agent` binary (or uses `ICEBERG_AGENT_BINARY` if provided)
- Uploads and installs `iceberg-agent` systemd service
- Ensures Docker Compose is available (`docker compose` plugin or `docker-compose`)
- Stores both agent endpoint IP (Tailscale) and public IP in config

## Agent endpoints

- `GET /health`
- `GET /status`
- `GET /metrics`
- `GET /services`
- `POST /services/{name}/deploy`
- `DELETE /services/{name}/destroy`
- `GET /services/{name}/logs`
- `POST /services/{name}/restart`
- `GET /dashboard/`
