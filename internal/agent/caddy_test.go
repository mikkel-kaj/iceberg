package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/compose"
)

func TestGenerateWriteCaddyfile(t *testing.T) {
	a := New("token", ":0", t.TempDir(), t.TempDir())
	body, err := a.GenerateCaddyfile([]CaddyEntry{{Domain: "domain.com", UpstreamHost: "svc", UpstreamPort: 3000}, {Domain: "other.com", UpstreamHost: "svc2", UpstreamPort: 8080}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "domain.com") || !strings.Contains(string(body), "reverse_proxy svc:3000") {
		t.Fatalf("unexpected caddyfile %s", string(body))
	}
	if err := a.WriteCaddyfile([]CaddyEntry{{Domain: "domain.com", UpstreamHost: "svc", UpstreamPort: 3000}}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(a.CaddyDir, "Caddyfile")
	first, _ := os.ReadFile(path)
	if err := a.WriteCaddyfile([]CaddyEntry{{Domain: "domain.com", UpstreamHost: "svc", UpstreamPort: 3000}}); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatal("expected idempotent output")
	}
}

func TestGetCaddyEntries(t *testing.T) {
	servicesDir := t.TempDir()
	a := New("token", ":0", servicesDir, t.TempDir())
	if err := a.writeService("my-app", compose.DeploySpec{Services: []compose.ServiceSpec{{Name: "my-app", Image: "nginx", Port: 80, Domain: "a.test.com"}}}); err != nil {
		t.Fatal(err)
	}
	entries, err := a.GetCaddyEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Domain != "a.test.com" {
		t.Fatalf("unexpected entries %#v", entries)
	}
}
