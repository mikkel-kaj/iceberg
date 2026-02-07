-- name: CreateServer :exec
INSERT INTO servers (
  name, tailscale_hostname, hetzner_id, ip, agent_token, ssh_key_id, firewall_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
);

-- name: GetServerByName :one
SELECT name, tailscale_hostname, hetzner_id, ip, agent_token, ssh_key_id, firewall_id, created_at
FROM servers
WHERE name = $1;

-- name: ListServers :many
SELECT name, tailscale_hostname, hetzner_id, ip, agent_token, ssh_key_id, firewall_id, created_at
FROM servers
ORDER BY created_at ASC;

-- name: DeleteServerByName :exec
DELETE FROM servers
WHERE name = $1;
