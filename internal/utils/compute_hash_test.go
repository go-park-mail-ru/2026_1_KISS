package utils

import (
	"testing"
)

func TestComputeHash_SameInput_SameHash(t *testing.T) {
	input := "hello world"
	h1 := ComputeHash(input)
	h2 := ComputeHash(input)
	if h1 != h2 {
		t.Errorf("expected same hash for same input, got %q and %q", h1, h2)
	}
}

func TestComputeHash_DifferentInputs_DifferentHashes(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{"different strings", "hello", "world"},
		{"empty vs non-empty", "", "a"},
		{"case sensitive", "Hello", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ha := ComputeHash(tt.a)
			hb := ComputeHash(tt.b)
			if ha == hb {
				t.Errorf("expected different hashes for %q and %q, both got %q", tt.a, tt.b, ha)
			}
		})
	}
}

func TestComputeHash_EmptyString(t *testing.T) {
	h := ComputeHash("")
	if h == "" {
		t.Error("expected non-empty hash for empty string")
	}
	if len(h) != 64 {
		t.Errorf("expected SHA-256 hex length 64, got %d", len(h))
	}
}
