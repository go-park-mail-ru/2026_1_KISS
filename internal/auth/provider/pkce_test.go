package provider

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateState_UniqueAndBase64URL(t *testing.T) {
	s1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState: %v", err)
	}
	s2, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState: %v", err)
	}
	if s1 == s2 {
		t.Fatalf("two states should differ: %q", s1)
	}
	if strings.ContainsAny(s1, "=+/") {
		t.Fatalf("state must be base64url (no =, +, /): %q", s1)
	}
	if _, err := base64.RawURLEncoding.DecodeString(s1); err != nil {
		t.Fatalf("state must be base64url-decodable: %v", err)
	}
}

func TestGenerateCodeVerifier_LengthAndCharset(t *testing.T) {
	v, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("GenerateCodeVerifier: %v", err)
	}
	if len(v) < 43 || len(v) > 128 {
		t.Fatalf("verifier length must be 43..128 per RFC 7636, got %d", len(v))
	}
	if strings.ContainsAny(v, "=+/") {
		t.Fatalf("verifier must be base64url (no =, +, /): %q", v)
	}
}

func TestCodeChallengeS256(t *testing.T) {
	verifier := "abc123"
	got := CodeChallengeS256(verifier)

	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])

	if got != want {
		t.Fatalf("S256 mismatch: got %q, want %q", got, want)
	}
	if strings.ContainsAny(got, "=+/") {
		t.Fatalf("challenge must be base64url-raw: %q", got)
	}
}
