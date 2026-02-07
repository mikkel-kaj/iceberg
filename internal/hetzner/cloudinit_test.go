package hetzner

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRenderCloudInit(t *testing.T) {
	out, err := RenderCloudInit(CloudInitParams{Hostname: "iceberg-01", SSHPublicKey: "ssh-ed25519 AAA", TailscaleKey: "tskey", AgentToken: "agt"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "iceberg-01") || !strings.Contains(out, "ssh-ed25519 AAA") {
		t.Fatal("missing expected content")
	}
	if !strings.Contains(out, "PasswordAuthentication no") || !strings.Contains(out, "docker-ce") {
		t.Fatal("missing required hardening/packages")
	}
	if strings.Contains(out, "docker.io") {
		t.Fatal("cloud-init should not install docker.io package")
	}
	if !strings.Contains(out, "download.docker.com/linux/ubuntu/gpg") {
		t.Fatal("missing Docker CE official repository bootstrap")
	}
	if !strings.Contains(out, "image: caddy:2") || !strings.Contains(out, "caddy-compose.yaml") {
		t.Fatal("missing Docker-based caddy bootstrap")
	}
	var parsed any
	if err := yaml.Unmarshal([]byte(strings.TrimPrefix(out, "#cloud-config\n")), &parsed); err != nil {
		t.Fatalf("invalid yaml: %v", err)
	}
}
