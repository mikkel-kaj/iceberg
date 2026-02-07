package cli

import (
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/config"
)

func TestServerTargetSelection(t *testing.T) {
	server := &config.ServerEntry{Name: "iceberg-01", IP: "100.64.0.10", PublicIP: "1.2.3.4"}
	if got := agentEndpointIP(server); got != "100.64.0.10" {
		t.Fatalf("unexpected agent endpoint ip %q", got)
	}
	if got := publicDNSIP(server); got != "1.2.3.4" {
		t.Fatalf("unexpected public dns ip %q", got)
	}
}

func TestServerTargetFallbacks(t *testing.T) {
	server := &config.ServerEntry{Name: "iceberg-02", PublicIP: "5.6.7.8"}
	if got := agentEndpointIP(server); got != "5.6.7.8" {
		t.Fatalf("unexpected fallback endpoint ip %q", got)
	}

	server = &config.ServerEntry{Name: "iceberg-03", IP: "100.64.0.11"}
	if got := publicDNSIP(server); got != "100.64.0.11" {
		t.Fatalf("unexpected fallback public ip %q", got)
	}
}

func TestAgentBaseURLValidation(t *testing.T) {
	if _, err := agentBaseURL(nil); err == nil {
		t.Fatal("expected nil server error")
	}
	if _, err := agentBaseURL(&config.ServerEntry{Name: "iceberg-04"}); err == nil {
		t.Fatal("expected missing ip error")
	}
}
