package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/config"
)

func TestServerCreateAndNaming(t *testing.T) {
	oldH := newHetznerClient
	defer func() { newHetznerClient = oldH }()
	mh := &mockHetzner{}
	newHetznerClient = func(token string) HetznerAPI { return mh }

	cfgPath := filepath.Join(t.TempDir(), ".iceberg", "config.yaml")
	cfg := &config.Config{HetznerToken: "token", TailscaleKey: "ts", Servers: []config.ServerEntry{{Name: "iceberg-01"}}}
	if err := cfg.Save(cfgPath); err != nil {
		t.Fatal(err)
	}

	cmd := newServerCreateCmd(&cfgPath)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Servers) != 2 || loaded.Servers[1].Name != "iceberg-02" {
		t.Fatalf("unexpected servers %#v", loaded.Servers)
	}
	if got := strings.Join(mh.calls, ","); got != "create_ssh,create_fw,create_server" {
		t.Fatalf("unexpected call order %s", got)
	}
}
