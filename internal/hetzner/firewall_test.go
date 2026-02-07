package hetzner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultFirewallRules(t *testing.T) {
	rules := DefaultFirewallRules()
	if len(rules) != 5 {
		t.Fatalf("expected 5 rules got %d", len(rules))
	}
	for _, rule := range rules {
		if rule.Direction != "in" {
			t.Fatalf("unexpected direction %s", rule.Direction)
		}
		if len(rule.SourceIPs) != 2 || rule.SourceIPs[0] != "0.0.0.0/0" {
			t.Fatalf("unexpected source ips %#v", rule.SourceIPs)
		}
	}
}

func TestCreateDeleteFirewall(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/firewalls":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["name"] != "iceberg-fw" {
				t.Fatalf("unexpected body %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"firewall":{"id":9,"name":"iceberg-fw"}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/firewalls/9":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s.Close()
	c := NewClient("token")
	c.BaseURL = s.URL
	fw, err := c.CreateFirewall(context.Background(), "iceberg-fw", DefaultFirewallRules())
	if err != nil {
		t.Fatal(err)
	}
	if fw.ID != 9 || fw.Name != "iceberg-fw" {
		t.Fatalf("unexpected firewall %#v", fw)
	}
	if err := c.DeleteFirewall(context.Background(), fw.ID); err != nil {
		t.Fatal(err)
	}
}
