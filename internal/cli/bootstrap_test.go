package cli

import "testing"

func TestFirstIPv4(t *testing.T) {
	ip, err := firstIPv4("Warning: test\n100.96.112.65\n")
	if err != nil {
		t.Fatal(err)
	}
	if ip != "100.96.112.65" {
		t.Fatalf("unexpected ip %q", ip)
	}
}

func TestFirstIPv4Missing(t *testing.T) {
	if _, err := firstIPv4("no ip here"); err == nil {
		t.Fatal("expected parsing error")
	}
}
