package agent

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectMetrics(t *testing.T) {
	m, err := CollectMetrics()
	if err != nil {
		t.Fatal(err)
	}
	if m == nil || m.MemoryTotalMB <= 0 || m.DiskTotalGB <= 0 {
		t.Fatalf("unexpected metrics %#v", m)
	}
	if m.CPUPercent < 0 || m.CPUPercent > 100 {
		t.Fatalf("cpu out of range %f", m.CPUPercent)
	}
	if m.MemoryPercent < 0 || m.MemoryPercent > 100 {
		t.Fatalf("mem out of range %f", m.MemoryPercent)
	}
}

func TestMetricsAndStatusHandlers(t *testing.T) {
	a := New("token", ":0", t.TempDir(), t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var m ServerMetrics
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}

	req = httptest.NewRequest(http.MethodGet, "/status", nil)
	w = httptest.NewRecorder()
	a.Router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var s StatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.Metrics == nil {
		t.Fatal("expected metrics in status")
	}
}
