package cli

import (
	"path/filepath"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/config"
)

func TestServerDestroyUsesTerraformProvisioner(t *testing.T) {
	oldTerraform := newTerraformClient
	oldTerraformBinaryPresent := terraformBinaryPresent
	defer func() {
		newTerraformClient = oldTerraform
		terraformBinaryPresent = oldTerraformBinaryPresent
	}()

	mt := &mockTerraform{}
	newTerraformClient = func() TerraformAPI { return mt }
	terraformBinaryPresent = func() bool { return true }

	cfgPath := filepath.Join(t.TempDir(), ".iceberg", "config.yaml")
	cfg := &config.Config{
		HetznerToken: "token",
		Servers: []config.ServerEntry{
			{Name: "iceberg-01", Provisioner: provisionerTerraform, TerraformDir: "/tmp/tf/iceberg-01"},
		},
	}
	if err := cfg.Save(cfgPath); err != nil {
		t.Fatal(err)
	}

	cmd := newServerDestroyCmd(&cfgPath)
	cmd.SetArgs([]string{"iceberg-01"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(mt.destroyCalls) != 1 {
		t.Fatalf("expected terraform destroy call, got %d", len(mt.destroyCalls))
	}
}

func TestServerDestroyRejectsNonTerraformProvisioner(t *testing.T) {
	oldTerraform := newTerraformClient
	oldTerraformBinaryPresent := terraformBinaryPresent
	defer func() {
		newTerraformClient = oldTerraform
		terraformBinaryPresent = oldTerraformBinaryPresent
	}()

	mt := &mockTerraform{}
	newTerraformClient = func() TerraformAPI { return mt }
	terraformBinaryPresent = func() bool { return true }

	cfgPath := filepath.Join(t.TempDir(), ".iceberg", "config.yaml")
	cfg := &config.Config{
		HetznerToken: "token",
		Servers: []config.ServerEntry{
			{Name: "legacy-01", Provisioner: "api"},
		},
	}
	if err := cfg.Save(cfgPath); err != nil {
		t.Fatal(err)
	}

	cmd := newServerDestroyCmd(&cfgPath)
	cmd.SetArgs([]string{"legacy-01"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected non-terraform rejection")
	}
	if len(mt.destroyCalls) != 0 {
		t.Fatalf("expected no terraform destroy calls, got %d", len(mt.destroyCalls))
	}
}
