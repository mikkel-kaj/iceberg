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
	newAgentClient = func(baseURL, token string) AgentAPI { return ma }
	newCloudflareClient = func(token string) CloudflareAPI { return &mockCloudflare{zoneID: "z1"} }

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{DefaultDomain: "status.test.com", Servers: []config.ServerEntry{{Name: "iceberg-01", IP: "100.64.0.1", AgentToken: "tok"}}}
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
	if !strings.Contains(buf.String(), "Manual DNS required") {
		t.Fatalf("expected manual dns message got %q", buf.String())
	}
}
