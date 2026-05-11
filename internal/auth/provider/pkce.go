package provider

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const (
	pkceVerifierBytes = 32
	stateBytes        = 32
)

func GenerateState() (string, error) {
	return randBase64URL(stateBytes)
}

func GenerateCodeVerifier() (string, error) {
	return randBase64URL(pkceVerifierBytes)
}

func CodeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randBase64URL(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
