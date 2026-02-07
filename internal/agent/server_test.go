package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthAndStatusPublic(t *testing.T) {
	a := New("token", ":0", t.TempDir(), t.TempDir())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var payload map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["status"] != "ok" {
		t.Fatalf("unexpected payload %#v", payload)
	}
}

func TestServicesEmptyAndUnknownRoute(t *testing.T) {
	a := New("token", ":0", t.TempDir(), t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if got := w.Body.String(); got != "[]\n" {
		t.Fatalf("expected [] got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/unknown", nil)
	w = httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestMutatingRoutesRequireAuth(t *testing.T) {
	a := New("token", ":0", t.TempDir(), t.TempDir())

	req := httptest.NewRequest(http.MethodPost, "/services/my-app/deploy", nil)
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/services/my-app/destroy", nil)
	w = httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/services/my-app/restart", nil)
	w = httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestStartStopLifecycle(t *testing.T) {
	a := New("token", "127.0.0.1:0", t.TempDir(), t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Start(ctx)
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	_ = a.Stop(context.Background())
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("unexpected start error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for start/stop")
	}
}
