package catalog

import (
	"strings"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/compose"
	"gopkg.in/yaml.v3"
)

func TestCatalogAllGet(t *testing.T) {
	all := All()
	if len(all) < 5 {
		t.Fatalf("expected >=5 entries got %d", len(all))
	}
	if _, err := Get("uptime-kuma"); err != nil {
		t.Fatal(err)
	}
	if _, err := Get("missing"); err == nil {
		t.Fatal("expected error")
	}
}

func TestUptimeKuma(t *testing.T) {
	e, _ := Get("uptime-kuma")
	if e.Port != 3001 || e.HealthPath != "/" {
		t.Fatalf("unexpected metadata %#v", e)
	}
	spec, err := e.BuildSpec("status.test.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Services) != 1 || spec.Services[0].Image != "louislam/uptime-kuma:1" {
		t.Fatalf("unexpected spec %#v", spec)
	}
}

func TestPlausible(t *testing.T) {
	e, _ := Get("plausible")
	spec, err := e.BuildSpec("analytics.test.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Services) != 3 {
		t.Fatalf("unexpected service count %d", len(spec.Services))
	}
	foundPostgres := false
	for _, s := range spec.Services {
		if strings.Contains(s.Image, "postgres") {
			foundPostgres = true
		}
	}
	if !foundPostgres {
		t.Fatal("expected postgres image")
	}
	if got := spec.Services[0].Env["SECRET_KEY_BASE"]; got == "" || got == "change-me" {
		t.Fatalf("expected generated SECRET_KEY_BASE, got %q", got)
	}
	override := "plausible-secret"
	specOverride, err := e.BuildSpec("analytics.test.com", map[string]string{"SECRET_KEY_BASE": override})
	if err != nil {
		t.Fatal(err)
	}
	if specOverride.Services[0].Env["SECRET_KEY_BASE"] != override {
		t.Fatal("expected SECRET_KEY_BASE override to be used")
	}
	mustComposeYAML(t, *spec)
}

func TestN8N(t *testing.T) {
	e, _ := Get("n8n")
	s1, err := e.BuildSpec("", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	k1 := s1.Services[0].Env["N8N_ENCRYPTION_KEY"]
	if k1 == "" {
		t.Fatal("expected auto key")
	}
	s2, err := e.BuildSpec("", map[string]string{"N8N_ENCRYPTION_KEY": "fixed"})
	if err != nil {
		t.Fatal(err)
	}
	if s2.Services[0].Env["N8N_ENCRYPTION_KEY"] != "fixed" {
		t.Fatal("expected override")
	}
	if len(s1.Services[0].Volumes) == 0 {
		t.Fatal("expected persistent volume")
	}
	mustComposeYAML(t, *s1)
}

func TestGitea(t *testing.T) {
	e, _ := Get("gitea")
	spec, err := e.BuildSpec("git.test.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Services) != 2 {
		t.Fatalf("unexpected count %d", len(spec.Services))
	}
	if len(spec.Services[0].DependsOn) == 0 {
		t.Fatal("expected depends_on")
	}
	mustComposeYAML(t, *spec)
}

func TestSupabase(t *testing.T) {
	e, _ := Get("supabase")
	spec, err := e.BuildSpec("db.test.com", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Services) < 4 {
		t.Fatalf("expected >=4 services got %d", len(spec.Services))
	}
	env := spec.Services[1].Env
	if env["JWT_SECRET"] == "" || env["ANON_KEY"] == "" || env["SERVICE_KEY"] == "" {
		t.Fatal("expected generated secrets")
	}
	if env["JWT_SECRET"] == env["ANON_KEY"] || env["ANON_KEY"] == env["SERVICE_KEY"] {
		t.Fatal("expected distinct secrets")
	}
	mustComposeYAML(t, *spec)
}

func mustComposeYAML(t *testing.T, spec compose.DeploySpec) {
	t.Helper()
	body, err := compose.GenerateComposeFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := yaml.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
}
