package agent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type logStreamerMock struct {
	serviceDir string
	container  string
	tail       int
	follow     bool
}

func (l *logStreamerMock) Read(_ context.Context, serviceDir, container string, tail int, follow bool) (io.ReadCloser, error) {
	l.serviceDir = serviceDir
	l.container = container
	l.tail = tail
	l.follow = follow
	return io.NopCloser(strings.NewReader("line1\nline2\n")), nil
}

func TestHandleLogs(t *testing.T) {
	servicesDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(servicesDir, "test"), 0o755)
	a := New("token", ":0", servicesDir, t.TempDir())
	ls := &logStreamerMock{}
	a.loger = ls
	req := httptest.NewRequest(http.MethodGet, "/services/test/logs?tail=10", nil)
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Fatalf("unexpected content type %s", ct)
	}
	if ls.container != "test" || ls.tail != 10 || ls.serviceDir == "" {
		t.Fatalf("unexpected log call: %#v", ls)
	}
}

func TestHandleLogsNotFound(t *testing.T) {
	a := New("token", ":0", t.TempDir(), t.TempDir())
	a.loger = &logStreamerMock{}
	req := httptest.NewRequest(http.MethodGet, "/services/missing/logs", nil)
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}
