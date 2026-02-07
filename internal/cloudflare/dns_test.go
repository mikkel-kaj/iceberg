package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetZoneID(t *testing.T) {
	var seenPath string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.String()
		_, _ = w.Write([]byte(`{"result":[{"id":"z1"}]}`))
	}))
	defer s.Close()
	c := NewClient("token")
	c.BaseURL = s.URL
	zone, err := c.GetZoneID(context.Background(), "analytics.mydomain.com")
	if err != nil {
		t.Fatal(err)
	}
	if zone != "z1" {
		t.Fatalf("unexpected zone %s", zone)
	}
	if !strings.Contains(seenPath, "name=mydomain.com") {
		t.Fatalf("unexpected path %s", seenPath)
	}
}

func TestCreateFindDeleteRecord(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/zones/z1/dns_records":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["type"] != "A" {
				t.Fatalf("unexpected body %#v", body)
			}
			_, _ = w.Write([]byte(`{"result":{"id":"r1","name":"a.test.com","type":"A","content":"1.2.3.4"}}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/zones/z1/dns_records"):
			_, _ = w.Write([]byte(`{"result":[{"id":"r1","name":"a.test.com","type":"A","content":"1.2.3.4"}]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/zones/z1/dns_records/r1":
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s.Close()
	c := NewClient("token")
	c.BaseURL = s.URL
	rec, err := c.CreateARecord(context.Background(), "z1", "a.test.com", "1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID != "r1" {
		t.Fatalf("unexpected rec %#v", rec)
	}
	found, err := c.FindRecord(context.Background(), "z1", "a.test.com")
	if err != nil {
		t.Fatal(err)
	}
	if found == nil || found.ID != "r1" {
		t.Fatalf("unexpected found %#v", found)
	}
	if err := c.DeleteRecord(context.Background(), "z1", "r1"); err != nil {
		t.Fatal(err)
	}
}

func TestCreateARecordUpdatesExisting(t *testing.T) {
	var putSeen bool
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/zones/z1/dns_records"):
			_, _ = w.Write([]byte(`{"result":[{"id":"r1","name":"a.test.com","type":"A","content":"1.2.3.4"}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/zones/z1/dns_records/r1":
			putSeen = true
			_, _ = w.Write([]byte(`{"result":{"id":"r1","name":"a.test.com","type":"A","content":"5.6.7.8"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s.Close()

	c := NewClient("token")
	c.BaseURL = s.URL
	rec, err := c.CreateARecord(context.Background(), "z1", "a.test.com", "5.6.7.8")
	if err != nil {
		t.Fatal(err)
	}
	if !putSeen {
		t.Fatal("expected update call")
	}
	if rec.Content != "5.6.7.8" {
		t.Fatalf("unexpected content %s", rec.Content)
	}
}
