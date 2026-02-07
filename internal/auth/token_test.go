package auth

import "testing"

func TestGenerateToken(t *testing.T) {
	a, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	b, err := GenerateToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 64 {
		t.Fatalf("expected 64 chars got %d", len(a))
	}
	if a == b {
		t.Fatal("expected random tokens")
	}
}

func TestValidateToken(t *testing.T) {
	if !ValidateToken("abc", "abc") {
		t.Fatal("expected true")
	}
	if ValidateToken("abc", "abd") {
		t.Fatal("expected false")
	}
	if ValidateToken("", "abc") {
		t.Fatal("expected false")
	}
}
