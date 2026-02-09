# Iceberg

Iceberg is a self-hosting control plane with:
- `iceberg` CLI as a thin wrapper around control APIs
- `iceberg control serve` local control agent (Terraform + Bitwarden + orchestration)
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
iceberg control serve
iceberg init
iceberg server create
iceberg server list
iceberg server destroy <name>
iceberg deploy uptime-kuma --domain status.example.com
iceberg deploy --image nginx:latest --name my-nginx --port 80
iceberg deploy --image my-api:latest --name my-api --port 8080 --env API_KEY=bw://item-id#field:API_KEY
iceberg status
iceberg logs <service>
iceberg destroy <service>
iceberg catalog
```

By default, the CLI calls the control agent at `http://127.0.0.1:19090` (`--control-url`).

`iceberg server create` now bootstraps the server automatically:
- Builds a Linux `iceberg-agent` binary (or uses `ICEBERG_AGENT_BINARY` if provided)
- Uploads and installs `iceberg-agent` systemd service
- Ensures Docker Compose is available (`docker compose` plugin or `docker-compose`)
- Stores both agent endpoint IP (Tailscale) and public IP in config
- Supports provisioners:
  - Terraform only for server lifecycle
  - Local `terraform` binary is required
  - State is stored in `~/.iceberg/terraform/<server-name>`

## Secret env values (Bitwarden CLI)

For `iceberg deploy`, env values can reference Bitwarden secrets:

```bash
iceberg deploy --image my-api:latest --name my-api --port 8080 \
  --env DB_PASSWORD=bw://<item-id> \
  --env DB_USER=bw://<item-id>#username \
  --env API_KEY=bw://<item-id>#field:API_KEY
```

Resolution happens in the control agent at deploy time using `bw` CLI, so plaintext secrets are not stored in config or compose templates.
`bw` must be installed and unlocked/authenticated in your current shell session.

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
