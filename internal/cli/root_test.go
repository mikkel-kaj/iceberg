package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelpAndVersion(t *testing.T) {
	cmd := NewRootCmd("v0.1.0-dev")
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, required := range []string{"init", "server", "deploy", "status", "destroy", "logs", "catalog"} {
		if !strings.Contains(out, required) {
			t.Fatalf("help missing %s", required)
		}
	}

	buf.Reset()
	cmd = NewRootCmd("v0.1.0-dev")
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "v0.1.0-dev") {
		t.Fatalf("unexpected version output %q", buf.String())
	}
}
