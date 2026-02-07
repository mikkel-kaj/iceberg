package hetzner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateGetDeleteServer(t *testing.T) {
	var createBody map[string]any
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/servers":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"server":{"id":11,"name":"iceberg-01","status":"running","public_net":{"ipv4":{"ip":"1.2.3.4"},"ipv6":{"ip":"::1"}}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/servers/11":
			_, _ = w.Write([]byte(`{"server":{"id":11,"name":"iceberg-01","status":"running","public_net":{"ipv4":{"ip":"1.2.3.4"},"ipv6":{"ip":"::1"}}}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/servers/11":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s.Close()

	c := NewClient("token")
	c.BaseURL = s.URL
	srv, err := c.CreateServer(context.Background(), CreateServerOpts{Name: "iceberg-01", SSHKeyIDs: []int64{1}, FirewallIDs: []int64{2}, UserData: "#cloud-config"})
	if err != nil {
		t.Fatal(err)
	}
	if createBody["server_type"] != "ccx13" || createBody["image"] != "ubuntu-24.04" || createBody["location"] != "nbg1" {
		t.Fatalf("defaults not applied: %#v", createBody)
	}
	if srv.IPv4 != "1.2.3.4" {
		t.Fatalf("unexpected ipv4 %s", srv.IPv4)
	}
	got, err := c.GetServer(context.Background(), srv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "running" {
		t.Fatalf("unexpected status %s", got.Status)
	}
	if err := c.DeleteServer(context.Background(), srv.ID); err != nil {
		t.Fatal(err)
	}
}
