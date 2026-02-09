package cli

import (
	"path/filepath"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/config"
)

func TestServerDestroyUsesTerraformProvisioner(t *testing.T) {
	oldTerraform := newTerraformClient
	oldTerraformBinaryPresent := terraformBinaryPresent
	oldHetzner := newHetznerClient
	defer func() {
		newTerraformClient = oldTerraform
		terraformBinaryPresent = oldTerraformBinaryPresent
		newHetznerClient = oldHetzner
	}()

	mt := &mockTerraform{}
	mh := &mockHetzner{}
	newTerraformClient = func() TerraformAPI { return mt }
	newHetznerClient = func(token string) HetznerAPI { return mh }
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
	if got := len(mh.calls); got != 0 {
		t.Fatalf("expected no hetzner calls, got %d", got)
	}
}
