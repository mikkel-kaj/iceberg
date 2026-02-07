package agent

import (
	"context"
	"errors"
	"testing"
)

func TestRunComposeFallbackToDockerComposeBinary(t *testing.T) {
	a := New("token", ":0", t.TempDir(), t.TempDir())
	rm := &runnerMock{errs: []error{errors.New("docker compose not available"), nil}}
	a.runner = rm

	if err := a.runCompose(context.Background(), t.TempDir(), "ps"); err != nil {
		t.Fatalf("expected fallback success, got %v", err)
	}
	if len(rm.calls) != 2 {
		t.Fatalf("expected 2 compose attempts, got %d", len(rm.calls))
	}
	if rm.calls[0].name != "docker" || len(rm.calls[0].args) < 1 || rm.calls[0].args[0] != "compose" {
		t.Fatalf("unexpected first call %#v", rm.calls[0])
	}
	if rm.calls[1].name != "docker-compose" {
		t.Fatalf("unexpected fallback call %#v", rm.calls[1])
	}
}

func TestRunComposeFailsWhenBothCommandsFail(t *testing.T) {
	a := New("token", ":0", t.TempDir(), t.TempDir())
	rm := &runnerMock{errs: []error{errors.New("missing plugin"), errors.New("missing binary")}}
	a.runner = rm

	if err := a.runCompose(context.Background(), t.TempDir(), "up", "-d"); err == nil {
		t.Fatal("expected compose failure")
	}
}
