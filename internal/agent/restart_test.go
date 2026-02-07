package agent

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleRestart(t *testing.T) {
	servicesDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(servicesDir, "my-app"), 0o755)
	a := New("token", ":0", servicesDir, t.TempDir())
	rm := &runnerMock{}
	a.runner = rm

	req := httptest.NewRequest(http.MethodPost, "/services/my-app/restart", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if len(rm.calls) == 0 || rm.calls[0].name != "docker" {
		t.Fatalf("expected docker command call, got %#v", rm.calls)
	}
}

func TestHandleRestartNotFound(t *testing.T) {
	a := New("token", ":0", t.TempDir(), t.TempDir())
	a.runner = &runnerMock{}
	req := httptest.NewRequest(http.MethodPost, "/services/missing/restart", nil)
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}
