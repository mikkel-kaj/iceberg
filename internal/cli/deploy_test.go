package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/config"
)

func TestDeployCustomValidation(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{Servers: []config.ServerEntry{{Name: "iceberg-01", IP: "100.64.0.1", AgentToken: "tok"}}}
	_ = cfg.Save(cfgPath)

	cmd := newDeployCmd(&cfgPath)
	cmd.SetArgs([]string{"--image", "nginx:latest"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--name") {
		t.Fatalf("unexpected err %v", err)
	}

	cmd = newDeployCmd(&cfgPath)
	cmd.SetArgs([]string{"--image", "nginx:latest", "--name", "my-nginx"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--port") {
		t.Fatalf("unexpected err %v", err)
	}
}

func TestDeployCatalogAndManualDNS(t *testing.T) {
	oldA, oldCF := newAgentClient, newCloudflareClient
	defer func() { newAgentClient, newCloudflareClient = oldA, oldCF }()
	ma := &mockAgent{}
	var baseURLSeen string
	newAgentClient = func(baseURL, token string) AgentAPI {
		baseURLSeen = baseURL
		return ma
	}
	newCloudflareClient = func(token string) CloudflareAPI { return &mockCloudflare{zoneID: "z1"} }

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{DefaultDomain: "status.test.com", Servers: []config.ServerEntry{{Name: "iceberg-01", IP: "100.64.0.1", PublicIP: "1.2.3.4", AgentToken: "tok"}}}
	_ = cfg.Save(cfgPath)

	cmd := newDeployCmd(&cfgPath)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"uptime-kuma"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if ma.deployName != "uptime-kuma" {
		t.Fatalf("unexpected deploy name %s", ma.deployName)
	}
	if baseURLSeen != "http://100.64.0.1:8443" {
		t.Fatalf("unexpected agent base url %q", baseURLSeen)
	}
	if !strings.Contains(buf.String(), "Manual DNS required: create A record status.test.com -> 1.2.3.4") {
		t.Fatalf("expected manual dns message got %q", buf.String())
	}
}
