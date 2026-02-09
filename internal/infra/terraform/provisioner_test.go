package terraform

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type runnerCall struct {
	dir  string
	env  map[string]string
	name string
	args []string
}

type runnerMock struct {
	calls []runnerCall
	outs  [][]byte
	errs  []error
}

func (m *runnerMock) Run(ctx context.Context, dir string, env map[string]string, name string, args ...string) ([]byte, error) {
	call := runnerCall{dir: dir, env: env, name: name, args: args}
	m.calls = append(m.calls, call)
	if len(m.outs) > 0 {
		out := m.outs[0]
		m.outs = m.outs[1:]
		var err error
		if len(m.errs) > 0 {
			err = m.errs[0]
			m.errs = m.errs[1:]
		}
		return out, err
	}
	if len(m.errs) > 0 {
		err := m.errs[0]
		m.errs = m.errs[1:]
		return nil, err
	}
	return nil, nil
}

func TestCreateServerWritesFilesAndRunsTerraform(t *testing.T) {
	rm := &runnerMock{
		outs: [][]byte{
			nil,
			nil,
			[]byte(`{"server_id":{"value":11},"server_ipv4":{"value":"1.2.3.4"},"ssh_key_id":{"value":12},"firewall_id":{"value":13}}`),
		},
	}
	p := &Provisioner{runner: rm}
	workDir := filepath.Join(t.TempDir(), "tf")
	result, err := p.CreateServer(context.Background(), CreateOptions{
		WorkDir:      workDir,
		Name:         "iceberg-01",
		SSHPublicKey: "ssh-ed25519 AAA",
		UserData:     "#cloud-config\nhostname: iceberg-01",
		Token:        "token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ServerID != 11 || result.IPv4 != "1.2.3.4" || result.SSHKeyID != 12 || result.FirewallID != 13 {
		t.Fatalf("unexpected result %#v", result)
	}
	mainTFBody, err := os.ReadFile(filepath.Join(workDir, "main.tf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainTFBody), `resource "hcloud_server" "iceberg"`) {
		t.Fatalf("main.tf missing expected resource")
	}
	tfvarsBody, err := os.ReadFile(filepath.Join(workDir, "terraform.tfvars.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tfvarsBody), `"name": "iceberg-01"`) {
		t.Fatalf("tfvars missing name: %s", string(tfvarsBody))
	}
	if len(rm.calls) != 3 {
		t.Fatalf("expected 3 terraform calls, got %d", len(rm.calls))
	}
	if rm.calls[0].name != "terraform" || rm.calls[0].args[0] != "init" {
		t.Fatalf("unexpected init call %#v", rm.calls[0])
	}
	if rm.calls[1].args[0] != "apply" || rm.calls[2].args[0] != "output" {
		t.Fatalf("unexpected call sequence %#v", rm.calls)
	}
	if rm.calls[0].env["HCLOUD_TOKEN"] != "token" {
		t.Fatalf("expected HCLOUD_TOKEN in env")
	}
}

func TestDestroyServerRunsTerraformDestroy(t *testing.T) {
	rm := &runnerMock{}
	p := &Provisioner{runner: rm}
	workDir := filepath.Join(t.TempDir(), "tf")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := p.DestroyServer(context.Background(), DestroyOptions{WorkDir: workDir, Token: "token"}); err != nil {
		t.Fatal(err)
	}
	if len(rm.calls) != 1 {
		t.Fatalf("expected single destroy call, got %d", len(rm.calls))
	}
	if rm.calls[0].args[0] != "destroy" {
		t.Fatalf("unexpected call %#v", rm.calls[0])
	}
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Fatalf("expected work dir removed, stat err=%v", err)
	}
}
