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
	var parsed any
	if err := yaml.Unmarshal([]byte(strings.TrimPrefix(out, "#cloud-config\n")), &parsed); err != nil {
		t.Fatalf("invalid yaml: %v", err)
	}
}
