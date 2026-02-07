package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mikkel-kaj/iceberg/internal/compose"
)

type containerCheckerMock struct {
	running map[string]bool
	restart map[string]int
	memory  map[string]int64
}

func (c *containerCheckerMock) IsRunning(name string) (bool, error)      { return c.running[name], nil }
func (c *containerCheckerMock) RestartCount(name string) (int, error)    { return c.restart[name], nil }
func (c *containerCheckerMock) MemoryUsageMB(name string) (int64, error) { return c.memory[name], nil }

type httpCheckerMock struct {
	code int
	err  error
}

func (h *httpCheckerMock) Check(_ string) (int, error) { return h.code, h.err }

func TestHealthMonitorStates(t *testing.T) {
	servicesDir := t.TempDir()
	mk := func(name, health string) {
		dir := filepath.Join(servicesDir, name)
		_ = os.MkdirAll(dir, 0o755)
		meta := serviceMeta{Name: name, HealthPath: health, Spec: compose.DeploySpec{Services: []compose.ServiceSpec{{Name: name, Port: 80}}}}
		raw, _ := json.Marshal(meta)
		_ = os.WriteFile(filepath.Join(dir, serviceMetaFile), raw, 0o644)
	}
	mk("ok", "/")
	mk("bad", "/")
	mk("stop", "/")
	cc := &containerCheckerMock{running: map[string]bool{"ok": true, "bad": true, "stop": false}, restart: map[string]int{"ok": 1, "bad": 1, "stop": 0}, memory: map[string]int64{"ok": 12, "bad": 13, "stop": 0}}
	hm := NewHealthMonitor(10*time.Millisecond, servicesDir, cc, &httpCheckerMock{code: http.StatusOK})
	hm.tick()
	s, err := hm.Get("ok")
	if err != nil || s.Status != HealthStatusHealthy {
		t.Fatalf("expected healthy got %#v err=%v", s, err)
	}
	hm.HTTPChecker = &httpCheckerMock{code: http.StatusInternalServerError}
	hm.tick()
	s, _ = hm.Get("bad")
	if s.Status != HealthStatusUnhealthy {
		t.Fatalf("expected unhealthy got %s", s.Status)
	}
	s, _ = hm.Get("stop")
	if s.Status != HealthStatusStopped {
		t.Fatalf("expected stopped got %s", s.Status)
	}
	cc.restart["ok"] = 5
	hm.tick()
	s, _ = hm.Get("ok")
	if s.Status != HealthStatusCrashLoop {
		t.Fatalf("expected crash-looping got %s", s.Status)
	}
	if len(hm.GetAll()) != 3 {
		t.Fatalf("expected 3 statuses got %d", len(hm.GetAll()))
	}
	if _, err := hm.Get("missing"); err == nil {
		t.Fatal("expected error")
	}
}

func TestHealthMonitorStart(t *testing.T) {
	servicesDir := t.TempDir()
	cc := &containerCheckerMock{running: map[string]bool{}, restart: map[string]int{}, memory: map[string]int64{}}
	hm := NewHealthMonitor(10*time.Millisecond, servicesDir, cc, &httpCheckerMock{code: http.StatusOK})
	ctx, cancel := context.WithCancel(context.Background())
	go hm.Start(ctx)
	time.Sleep(30 * time.Millisecond)
	cancel()
}
