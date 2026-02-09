package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestControlClientCreateServer(t *testing.T) {
	var seenCfg string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/servers" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		seenCfg = body["config_path"]
		_ = json.NewEncoder(w).Encode(ServerCreateResult{Name: "iceberg-01", PublicIP: "1.2.3.4", AgentIP: "100.64.0.10", Provisioner: provisionerTerraform})
	}))
	defer s.Close()

	c := newDefaultControlClient(s.URL)
	res, err := c.CreateServer(context.Background(), "/tmp/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if res.Name != "iceberg-01" || seenCfg != "/tmp/config.yaml" {
		t.Fatalf("unexpected result=%#v seenCfg=%q", res, seenCfg)
	}
}

func TestControlClientDeploy(t *testing.T) {
	var seenRef string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/deploy" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct {
			ConfigPath string      `json:"config_path"`
			Input      DeployInput `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		seenRef = body.Input.Env["API_KEY"]
		_ = json.NewEncoder(w).Encode(DeployResult{ServiceName: "my-nginx", ServerName: "iceberg-01"})
	}))
	defer s.Close()

	c := newDefaultControlClient(s.URL)
	res, err := c.Deploy(context.Background(), "/tmp/config.yaml", DeployInput{
		Image: "nginx:latest",
		Name:  "my-nginx",
		Port:  80,
		Env:   map[string]string{"API_KEY": "bw://item-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ServiceName != "my-nginx" {
		t.Fatalf("unexpected deploy response %#v", res)
	}
	if seenRef != "bw://item-1" {
		t.Fatalf("expected raw ref, got %q", seenRef)
	}
}

func TestControlClientError(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer s.Close()

	c := newDefaultControlClient(s.URL)
	_, err := c.CreateServer(context.Background(), "/tmp/config.yaml")
	if err == nil || !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("unexpected error %v", err)
	}
}
