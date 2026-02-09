package cli

import (
	"bytes"
	"testing"
)

func TestServerCreateUsesControlClient(t *testing.T) {
	oldControl := newControlClient
	defer func() { newControlClient = oldControl }()
	mc := &mockControl{}
	newControlClient = func(baseURL string) ControlAPI { return mc }

	cfgPath := "/tmp/config.yaml"
	controlURL := "http://127.0.0.1:19090"
	cmd := newServerCreateCmdWithControl(&cfgPath, &controlURL)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if mc.createCfg != cfgPath {
		t.Fatalf("unexpected cfg path %q", mc.createCfg)
	}
}

func TestServerDestroyUsesControlClient(t *testing.T) {
	oldControl := newControlClient
	defer func() { newControlClient = oldControl }()
	mc := &mockControl{}
	newControlClient = func(baseURL string) ControlAPI { return mc }

	cfgPath := "/tmp/config.yaml"
	controlURL := "http://127.0.0.1:19090"
	cmd := newServerDestroyCmdWithControl(&cfgPath, &controlURL)
	cmd.SetArgs([]string{"iceberg-01"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if mc.destroyCfg != cfgPath || mc.destroyName != "iceberg-01" {
		t.Fatalf("unexpected destroy args cfg=%q name=%q", mc.destroyCfg, mc.destroyName)
	}
}

func TestDeployUsesControlClient(t *testing.T) {
	oldControl := newControlClient
	defer func() { newControlClient = oldControl }()
	mc := &mockControl{}
	newControlClient = func(baseURL string) ControlAPI { return mc }

	cfgPath := "/tmp/config.yaml"
	controlURL := "http://127.0.0.1:19090"
	cmd := newDeployCmdWithControl(&cfgPath, &controlURL)
	cmd.SetArgs([]string{"--image", "nginx:latest", "--name", "my-nginx", "--port", "80", "--env", "API_KEY=bw://item-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if mc.deployCfg != cfgPath {
		t.Fatalf("unexpected cfg path %q", mc.deployCfg)
	}
	if got := mc.deployIn.Env["API_KEY"]; got != "bw://item-1" {
		t.Fatalf("expected raw bitwarden ref, got %q", got)
	}
}
