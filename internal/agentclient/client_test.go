package agentclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mikkel-kaj/iceberg/internal/compose"
)

func TestDeployAndHeaders(t *testing.T) {
	var seenAuth, seenPath, seenDomain string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		seenPath = r.URL.Path
		var spec compose.DeploySpec
		_ = json.NewDecoder(r.Body).Decode(&spec)
		if len(spec.Services) > 0 {
			seenDomain = spec.Services[0].Domain
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer s.Close()
	c := NewClient(s.URL, "tok")
	err := c.Deploy(context.Background(), "my-app", compose.DeploySpec{Services: []compose.ServiceSpec{{Name: "my-app", Image: "nginx", Port: 80}}}, "status.test.com")
	if err != nil {
		t.Fatal(err)
	}
	if seenAuth != "Bearer tok" || seenPath != "/services/my-app/deploy" || seenDomain != "status.test.com" {
		t.Fatalf("unexpected auth/path/domain %s %s %s", seenAuth, seenPath, seenDomain)
	}
}

func TestDeployError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad"))
	}))
	defer s.Close()
	c := NewClient(s.URL, "tok")
	if err := c.Deploy(context.Background(), "my-app", compose.DeploySpec{Services: []compose.ServiceSpec{{Name: "my-app", Image: "nginx"}}}, ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestDestroyStatusLogs(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/services/my-app/destroy":
			w.WriteHeader(http.StatusOK)
		case "/status":
			_, _ = w.Write([]byte(`{"metrics":{"cpu_percent":1},"services":[]}`))
		case "/services/my-app/logs":
			_, _ = w.Write([]byte("line"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s.Close()
	c := NewClient(s.URL, "tok")
	if err := c.Destroy(context.Background(), "my-app"); err != nil {
		t.Fatal(err)
	}
	st, err := c.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.Metrics == nil {
		t.Fatal("expected metrics")
	}
	rc, err := c.Logs(context.Background(), "my-app")
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	body, _ := io.ReadAll(rc)
	if strings.TrimSpace(string(body)) != "line" {
		t.Fatalf("unexpected logs %q", string(body))
	}
}
