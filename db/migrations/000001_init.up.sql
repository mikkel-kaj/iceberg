CREATE TABLE IF NOT EXISTS servers (
    name TEXT PRIMARY KEY,
    tailscale_hostname TEXT NOT NULL,
    hetzner_id BIGINT NOT NULL,
    ip TEXT NOT NULL,
    agent_token TEXT NOT NULL,
    ssh_key_id BIGINT NOT NULL DEFAULT 0,
    firewall_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS services (
    name TEXT PRIMARY KEY,
    server_name TEXT NOT NULL REFERENCES servers(name) ON DELETE CASCADE,
    domain TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'unknown',
    spec_json TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_services_server_name ON services(server_name);
