package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func ComputeHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
