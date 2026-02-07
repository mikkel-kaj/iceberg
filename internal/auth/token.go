package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
)

func GenerateToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func ValidateToken(provided, expected string) bool {
	if provided == "" || expected == "" {
		return false
	}
	if len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
