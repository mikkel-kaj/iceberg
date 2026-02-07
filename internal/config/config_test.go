package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNonExistent(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSaveCreatesDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".iceberg", "config.yaml")
	cfg := &Config{HetznerToken: "h", TailscaleKey: "t"}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".iceberg", "config.yaml")
	cfg := &Config{
		HetznerToken:  "h",
		TailscaleKey:  "t",
		DefaultDomain: "test.com",
		DNS: &DNSConfig{
			Provider:        "cloudflare",
			CloudflareToken: "cf",
		},
		Servers: []ServerEntry{{Name: "iceberg-01", IP: "100.64.0.10", PublicIP: "1.2.3.4", AgentToken: "a", HetznerID: 42}},
	}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.HetznerToken != cfg.HetznerToken || loaded.DNS.Provider != "cloudflare" || len(loaded.Servers) != 1 {
		t.Fatalf("unexpected load result: %#v", loaded)
	}
	if loaded.Servers[0].PublicIP != "1.2.3.4" || loaded.Servers[0].IP != "100.64.0.10" {
		t.Fatalf("unexpected server ip fields %#v", loaded.Servers[0])
	}
}

func TestSaveFileMode0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &Config{HetznerToken: "h", TailscaleKey: "t"}
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("expected 0600 got %o", st.Mode().Perm())
	}
}

func TestServerHelpers(t *testing.T) {
	cfg := &Config{Servers: []ServerEntry{{Name: "a"}, {Name: "b"}}}
	cfg.AddServer(ServerEntry{Name: "c"})
	if len(cfg.Servers) != 3 {
		t.Fatalf("expected 3 got %d", len(cfg.Servers))
	}
	cfg.RemoveServer("b")
	if len(cfg.Servers) != 2 {
		t.Fatalf("expected 2 got %d", len(cfg.Servers))
	}
	if _, err := cfg.GetServer("missing"); err == nil {
		t.Fatal("expected error")
	}
	s, err := cfg.GetServer("c")
	if err != nil || s.Name != "c" {
		t.Fatalf("unexpected server: %#v err=%v", s, err)
	}
}
