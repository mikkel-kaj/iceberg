package secrets

import (
	"context"
	"errors"
	"testing"
)

func TestResolveBitwardenReferencePassword(t *testing.T) {
	oldRun := runCommand
	defer func() { runCommand = oldRun }()
	runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(`{"login":{"password":"super-secret"}}`), nil
	}

	secret, err := ResolveBitwardenReference(context.Background(), "bw://item-123")
	if err != nil {
		t.Fatal(err)
	}
	if secret != "super-secret" {
		t.Fatalf("unexpected secret %q", secret)
	}
}

func TestResolveBitwardenReferenceCustomField(t *testing.T) {
	oldRun := runCommand
	defer func() { runCommand = oldRun }()
	runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte(`{"fields":[{"name":"API_KEY","value":"abc"}]}`), nil
	}

	secret, err := ResolveBitwardenReference(context.Background(), "bitwarden://item-123#field:API_KEY")
	if err != nil {
		t.Fatal(err)
	}
	if secret != "abc" {
		t.Fatalf("unexpected secret %q", secret)
	}
}

func TestResolveBitwardenReferenceFallbackWithoutRaw(t *testing.T) {
	oldRun := runCommand
	defer func() { runCommand = oldRun }()
	call := 0
	runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		call++
		if call == 1 {
			return []byte("unknown flag: --raw"), errors.New("flag error")
		}
		return []byte(`{"login":{"password":"ok"}}`), nil
	}

	secret, err := ResolveBitwardenReference(context.Background(), "bw://item-123#password")
	if err != nil {
		t.Fatal(err)
	}
	if secret != "ok" {
		t.Fatalf("unexpected secret %q", secret)
	}
}

func TestResolveBitwardenReferenceInvalid(t *testing.T) {
	if _, err := ResolveBitwardenReference(context.Background(), "plain-secret"); err == nil {
		t.Fatal("expected invalid reference error")
	}
}
