package hetzner

import (
	"bytes"
	"errors"
	"text/template"
)

type CloudInitParams struct {
	Hostname     string
	SSHPublicKey string
	TailscaleKey string
	AgentToken   string
}

const cloudInitTemplate = `#cloud-config
hostname: {{ .Hostname }}
users:
  - name: iceberg
    shell: /bin/bash
    groups: [sudo, docker]
    sudo: ["ALL=(ALL) NOPASSWD:ALL"]
    ssh_authorized_keys:
      - {{ .SSHPublicKey }}
package_update: true
package_upgrade: true
packages:
  - ca-certificates
  - curl
write_files:
  - path: /etc/ssh/sshd_config.d/99-iceberg.conf
    permissions: "0644"
    content: |
      PasswordAuthentication no
  - path: /etc/iceberg-agent.env
    permissions: "0600"
    content: |
      ICEBERG_AGENT_TOKEN={{ .AgentToken }}
      TAILSCALE_AUTH_KEY={{ .TailscaleKey }}
  - path: /tmp/caddy-compose.yaml
    permissions: "0644"
    content: |
      services:
        caddy:
          image: caddy:2
          container_name: iceberg-caddy
          restart: unless-stopped
          ports:
            - "80:80"
            - "443:443"
            - "443:443/udp"
          extra_hosts:
            - "host.docker.internal:host-gateway"
          volumes:
            - ./Caddyfile:/etc/caddy/Caddyfile:ro
            - ./caddy_data:/data
            - ./caddy_config:/config
          logging:
            driver: local
  - path: /tmp/Caddyfile
    permissions: "0644"
    content: |
      # Managed by iceberg-agent
runcmd:
  - mkdir -p /etc/apt/keyrings
  - curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  - chmod a+r /etc/apt/keyrings/docker.asc
  - echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo \"$VERSION_CODENAME\") stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
  - apt-get update
  - apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  - usermod -aG docker iceberg
  - curl -fsSL https://tailscale.com/install.sh | sh
  - tailscale up --authkey={{ .TailscaleKey }} --ssh --hostname={{ .Hostname }}
  - systemctl restart ssh
  - systemctl enable --now docker
  - systemctl disable --now caddy || true
  - |
    if ! docker compose version >/dev/null 2>&1; then
      if ! command -v docker-compose >/dev/null 2>&1; then
        arch="$(uname -m)"
        case "$arch" in
          x86_64|amd64) compose_arch="x86_64" ;;
          aarch64|arm64) compose_arch="aarch64" ;;
          *) echo "unsupported architecture for docker compose: $arch" >&2; exit 1 ;;
        esac
        mkdir -p /usr/local/lib/docker/cli-plugins
        curl -fsSL "https://github.com/docker/compose/releases/download/v2.29.7/docker-compose-linux-${compose_arch}" -o /usr/local/lib/docker/cli-plugins/docker-compose
        chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
      fi
    fi
  - mkdir -p /opt/iceberg/services /opt/iceberg/caddy/caddy_data /opt/iceberg/caddy/caddy_config
  - mv /tmp/caddy-compose.yaml /opt/iceberg/caddy/caddy-compose.yaml
  - mv /tmp/Caddyfile /opt/iceberg/caddy/Caddyfile
  - cd /opt/iceberg/caddy && if docker compose version >/dev/null 2>&1; then docker compose -f caddy-compose.yaml up -d; else docker-compose -f caddy-compose.yaml up -d; fi
`

func RenderCloudInit(params CloudInitParams) (string, error) {
	if params.Hostname == "" || params.SSHPublicKey == "" || params.TailscaleKey == "" || params.AgentToken == "" {
		return "", errors.New("hostname, ssh key, tailscale key and agent token are required")
	}
	tmpl, err := template.New("cloudinit").Parse(cloudInitTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params); err != nil {
		return "", err
	}
	return buf.String(), nil
}
