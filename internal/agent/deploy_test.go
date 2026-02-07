package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/compose"
)

type runnerMock struct {
	calls []runnerCall
	err   error
	errs  []error
}

type runnerCall struct {
	dir  string
	name string
	args []string
}

func (r *runnerMock) Run(_ context.Context, dir string, name string, args ...string) error {
	r.calls = append(r.calls, runnerCall{dir: dir, name: name, args: args})
	if len(r.errs) > 0 {
		next := r.errs[0]
		r.errs = r.errs[1:]
		return next
	}
	return r.err
}

func TestHandleDeployAndDestroy(t *testing.T) {
	servicesDir := t.TempDir()
	caddyDir := t.TempDir()
	a := New("token", ":0", servicesDir, caddyDir)
	rm := &runnerMock{}
	a.runner = rm

	spec := compose.DeploySpec{Services: []compose.ServiceSpec{{Name: "my-app", Image: "nginx:latest", Port: 80}}}
	body, _ := json.Marshal(spec)
	req := httptest.NewRequest(http.MethodPost, "/services/my-app/deploy", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	composePath := filepath.Join(servicesDir, "my-app", composeFileName)
	if _, err := os.Stat(composePath); err != nil {
		t.Fatalf("expected compose file: %v", err)
	}

	req = httptest.NewRequest(http.MethodDelete, "/services/my-app/destroy", nil)
	req.Header.Set("Authorization", "Bearer token")
	w = httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if _, err := os.Stat(filepath.Join(servicesDir, "my-app")); !os.IsNotExist(err) {
		t.Fatalf("expected directory removed, stat err=%v", err)
	}
}

func TestHandleDeployValidation(t *testing.T) {
	a := New("token", ":0", t.TempDir(), t.TempDir())
	a.runner = &runnerMock{}
	spec := compose.DeploySpec{Services: []compose.ServiceSpec{{Name: "my-app", Image: ""}}}
	body, _ := json.Marshal(spec)
	req := httptest.NewRequest(http.MethodPost, "/services/my-app/deploy", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}
