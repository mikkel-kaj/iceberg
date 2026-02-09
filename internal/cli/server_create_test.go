package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/config"
)

func TestServerCreateAndNaming(t *testing.T) {
	oldH := newHetznerClient
	oldBootstrap := bootstrapServer
	oldTerraform := newTerraformClient
	oldTerraformBinaryPresent := terraformBinaryPresent
	defer func() {
		newHetznerClient = oldH
		bootstrapServer = oldBootstrap
		newTerraformClient = oldTerraform
		terraformBinaryPresent = oldTerraformBinaryPresent
	}()
	mh := &mockHetzner{}
	newHetznerClient = func(token string) HetznerAPI { return mh }
	newTerraformClient = func() TerraformAPI { return &mockTerraform{} }
	terraformBinaryPresent = func() bool { return false }
	bootstrapServer = func(ctx context.Context, serverName, publicIP, agentToken string, privateKey []byte) (string, error) {
		return "100.64.0.10", nil
	}

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

func TestServerCreateWithTerraformProvisioner(t *testing.T) {
	oldH := newHetznerClient
	oldBootstrap := bootstrapServer
	oldTerraform := newTerraformClient
	oldTerraformBinaryPresent := terraformBinaryPresent
	defer func() {
		newHetznerClient = oldH
		bootstrapServer = oldBootstrap
		newTerraformClient = oldTerraform
		terraformBinaryPresent = oldTerraformBinaryPresent
	}()
	mh := &mockHetzner{}
	mt := &mockTerraform{}
	newHetznerClient = func(token string) HetznerAPI { return mh }
	newTerraformClient = func() TerraformAPI { return mt }
	terraformBinaryPresent = func() bool { return true }
	bootstrapServer = func(ctx context.Context, serverName, publicIP, agentToken string, privateKey []byte) (string, error) {
		return "100.64.0.11", nil
	}

	cfgPath := filepath.Join(t.TempDir(), ".iceberg", "config.yaml")
	cfg := &config.Config{HetznerToken: "token", TailscaleKey: "ts"}
	if err := cfg.Save(cfgPath); err != nil {
		t.Fatal(err)
	}

	cmd := newServerCreateCmd(&cfgPath)
	cmd.SetArgs([]string{"--provisioner", "terraform"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Servers) != 1 {
		t.Fatalf("expected one server, got %d", len(loaded.Servers))
	}
	s := loaded.Servers[0]
	if s.Provisioner != provisionerTerraform {
		t.Fatalf("expected terraform provisioner, got %q", s.Provisioner)
	}
	if s.IP != "100.64.0.11" || s.PublicIP != "1.2.3.4" {
		t.Fatalf("unexpected ip values %#v", s)
	}
	if len(mt.createCalls) != 1 {
		t.Fatalf("expected terraform create call, got %d", len(mt.createCalls))
	}
	if got := strings.Join(mh.calls, ","); got != "" {
		t.Fatalf("expected no direct hetzner api calls, got %s", got)
	}
}
