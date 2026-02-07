package hetzner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGenerateSSHKeyPair(t *testing.T) {
	pub1, priv1, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	pub2, priv2, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(pub1, "ssh-ed25519") {
		t.Fatalf("unexpected pub key: %s", pub1)
	}
	if len(priv1) == 0 {
		t.Fatal("private key empty")
	}
	if pub1 == pub2 || string(priv1) == string(priv2) {
		t.Fatal("keys should differ")
	}
}

func TestCreateDeleteSSHKey(t *testing.T) {
	var createSeen, deleteSeen bool
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("missing auth header: %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/ssh_keys":
			createSeen = true
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["name"] != "iceberg" || !strings.HasPrefix(body["public_key"], "ssh-ed25519") {
				t.Fatalf("unexpected body %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"ssh_key":{"id":7,"name":"iceberg","fingerprint":"fp"}}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/ssh_keys/7":
			deleteSeen = true
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s.Close()

	c := NewClient("token")
	c.BaseURL = s.URL
	pub, _, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	key, err := c.CreateSSHKey(context.Background(), "iceberg", pub)
	if err != nil {
		t.Fatal(err)
	}
	if key.ID != 7 || key.Fingerprint != "fp" {
		t.Fatalf("unexpected key %#v", key)
	}
	if err := c.DeleteSSHKey(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if !createSeen || !deleteSeen {
		t.Fatalf("createSeen=%v deleteSeen=%v", createSeen, deleteSeen)
	}
}

func TestCreateSSHKeyErrors(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ssh_keys" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("bad token"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()
	c := NewClient("token")
	c.BaseURL = s.URL
	if _, err := c.CreateSSHKey(context.Background(), "iceberg", "ssh-ed25519 AAA"); err == nil {
		t.Fatal("expected error")
	}
}
