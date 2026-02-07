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
  - docker.io
  - docker-compose-plugin
  - caddy
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
runcmd:
  - usermod -aG docker iceberg
  - curl -fsSL https://tailscale.com/install.sh | sh
  - tailscale up --authkey={{ .TailscaleKey }} --ssh
  - systemctl restart ssh
  - systemctl enable --now docker
  - mkdir -p /opt/iceberg/services /opt/iceberg/caddy
  - echo "docker-ce" >/opt/iceberg/docker-marker
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
