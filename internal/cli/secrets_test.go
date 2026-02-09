package cli

import (
	"context"
	"strings"
	"testing"
)

func TestResolveEnvSecretsNoRefs(t *testing.T) {
	env, err := resolveEnvSecrets(context.Background(), map[string]string{"A": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if env["A"] != "1" {
		t.Fatalf("unexpected value %q", env["A"])
	}
}

func TestResolveEnvSecretsBitwarden(t *testing.T) {
	oldResolve := resolveBitwardenSecret
	defer func() { resolveBitwardenSecret = oldResolve }()
	resolveBitwardenSecret = func(ctx context.Context, ref string) (string, error) {
		if ref != "bw://item-1" {
			t.Fatalf("unexpected ref %q", ref)
		}
		return "secret-value", nil
	}

	env, err := resolveEnvSecrets(context.Background(), map[string]string{"TOKEN": "bw://item-1"})
	if err != nil {
		t.Fatal(err)
	}
	if env["TOKEN"] != "secret-value" {
		t.Fatalf("unexpected resolved value %q", env["TOKEN"])
	}
}

func TestResolveEnvSecretsBitwardenError(t *testing.T) {
	oldResolve := resolveBitwardenSecret
	defer func() { resolveBitwardenSecret = oldResolve }()
	resolveBitwardenSecret = func(ctx context.Context, ref string) (string, error) {
		return "", context.DeadlineExceeded
	}

	_, err := resolveEnvSecrets(context.Background(), map[string]string{"TOKEN": "bitwarden://item-1"})
	if err == nil || !strings.Contains(err.Error(), "resolve TOKEN") {
		t.Fatalf("unexpected error %v", err)
	}
}
