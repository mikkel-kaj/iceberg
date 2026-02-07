package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/agent"
	"github.com/mikkel-kaj/iceberg/internal/agentclient"
	"github.com/mikkel-kaj/iceberg/internal/cloudflare"
	"github.com/mikkel-kaj/iceberg/internal/config"
)

func TestStatusDestroyLogsCatalog(t *testing.T) {
	oldA, oldCF := newAgentClient, newCloudflareClient
	defer func() { newAgentClient, newCloudflareClient = oldA, oldCF }()
	ma := &mockAgent{statusResp: &agentclient.StatusResponse{Metrics: &agent.ServerMetrics{CPUPercent: 10, MemoryUsedMB: 100, MemoryTotalMB: 200, DiskUsedGB: 1, DiskTotalGB: 10}, Services: []agent.ServiceHealth{}}}
	newAgentClient = func(baseURL, token string) AgentAPI { return ma }
	newCloudflareClient = func(token string) CloudflareAPI {
		return &mockCloudflare{zoneID: "z1", record: &cloudflare.DNSRecord{ID: "r1"}}
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{DefaultDomain: "status.test.com", DNS: &config.DNSConfig{Provider: "cloudflare", CloudflareToken: "cf"}, Servers: []config.ServerEntry{{Name: "iceberg-01", IP: "100.64.0.1", AgentToken: "tok"}}}
	_ = cfg.Save(cfgPath)

	status := newStatusCmd(&cfgPath)
	buf := &bytes.Buffer{}
	status.SetOut(buf)
	if err := status.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No services deployed.") {
		t.Fatalf("unexpected status output %q", buf.String())
	}

	destroy := newDestroyCmd(&cfgPath)
	buf.Reset()
	destroy.SetOut(buf)
	destroy.SetArgs([]string{"my-app", "--domain", "status.test.com"})
	if err := destroy.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Destroyed service my-app") {
		t.Fatalf("unexpected destroy output %q", buf.String())
	}

	logs := newLogsCmd(&cfgPath)
	buf.Reset()
	logs.SetOut(buf)
	logs.SetArgs([]string{"my-app"})
	if err := logs.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "line") {
		t.Fatalf("unexpected logs output %q", buf.String())
	}

	catalogCmd := newCatalogCmd()
	buf.Reset()
	catalogCmd.SetOut(buf)
	if err := catalogCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"uptime-kuma", "plausible", "n8n", "gitea", "supabase"} {
		if !strings.Contains(buf.String(), name) {
			t.Fatalf("catalog output missing %s", name)
		}
	}
}
