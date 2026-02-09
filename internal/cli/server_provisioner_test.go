package cli

import "testing"

func TestChooseProvisionerAutoFallback(t *testing.T) {
	old := terraformBinaryPresent
	defer func() { terraformBinaryPresent = old }()
	terraformBinaryPresent = func() bool { return false }

	got, err := chooseProvisioner("auto")
	if err != nil {
		t.Fatal(err)
	}
	if got != provisionerAPI {
		t.Fatalf("expected api fallback, got %q", got)
	}
}

func TestChooseProvisionerTerraformRequiresBinary(t *testing.T) {
	old := terraformBinaryPresent
	defer func() { terraformBinaryPresent = old }()
	terraformBinaryPresent = func() bool { return false }

	if _, err := chooseProvisioner("terraform"); err == nil {
		t.Fatal("expected terraform missing error")
	}
}
