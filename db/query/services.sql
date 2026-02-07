-- name: UpsertService :exec
INSERT INTO services (
  name, server_name, domain, status, spec_json
) VALUES (
  $1, $2, $3, $4, $5
)
ON CONFLICT (name)
DO UPDATE SET
  server_name = excluded.server_name,
  domain = excluded.domain,
  status = excluded.status,
  spec_json = excluded.spec_json,
  updated_at = NOW();

-- name: GetServiceByName :one
SELECT name, server_name, domain, status, spec_json, created_at, updated_at
FROM services
WHERE name = $1;

-- name: ListServices :many
SELECT name, server_name, domain, status, spec_json, created_at, updated_at
FROM services
ORDER BY created_at ASC;

-- name: DeleteServiceByName :exec
DELETE FROM services
WHERE name = $1;
