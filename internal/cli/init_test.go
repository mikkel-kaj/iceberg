package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/config"
)

func TestInitWritesConfig(t *testing.T) {
	oldH, oldCF := newHetznerClient, newCloudflareClient
	defer func() { newHetznerClient, newCloudflareClient = oldH, oldCF }()

	h := &mockHetzner{}
	cf := &mockCloudflare{zoneID: "z1"}
	newHetznerClient = func(token string) HetznerAPI { return h }
	newCloudflareClient = func(token string) CloudflareAPI { return cf }

	cfgPath := filepath.Join(t.TempDir(), ".iceberg", "config.yaml")
	cmd := newInitCmd(&cfgPath)
	in := bytes.NewBufferString("h-token\nts-key\nexample.com\ncf-token\n")
	out := &bytes.Buffer{}
	cmd.SetIn(in)
	cmd.SetOut(out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HetznerToken != "h-token" || cfg.DNS == nil || cfg.DNS.CloudflareToken != "cf-token" {
		t.Fatalf("unexpected config %#v", cfg)
	}
}

func TestInitInvalidHetzner(t *testing.T) {
	oldH := newHetznerClient
	defer func() { newHetznerClient = oldH }()
	newHetznerClient = func(token string) HetznerAPI { return &mockHetzner{validateErr: os.ErrPermission} }

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cmd := newInitCmd(&cfgPath)
	cmd.SetIn(bytes.NewBufferString("h\nt\n\n"))
	cmd.SetOut(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid Hetzner token") {
		t.Fatalf("unexpected err %v", err)
	}
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("expected no config file, stat err=%v", err)
	}
}
