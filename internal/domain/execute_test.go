package domain

import (
	"errors"
	"testing"
	"time"
)

func TestBlockExecutionResult_String_Nil(t *testing.T) {
	var r *BlockExecutionResult
	result := r.String()
	if result != "<nil>" {
		t.Fatalf("expected '<nil>', got %q", result)
	}
}

func TestBlockExecutionResult_String_WithError(t *testing.T) {
	now := time.Now()
	r := &BlockExecutionResult{
		BlockID:    1,
		Position:   0,
		ExecutedAt: now,
		Duration:   time.Second,
		Error:      errors.New("test error"),
	}
	result := r.String()
	if result == "" {
		t.Fatal("expected non-empty string")
	}
	if !Contains(result, "Error: test error") {
		t.Fatalf("expected error message in result: %s", result)
	}
}

func TestBlockExecutionResult_String_WithOutput(t *testing.T) {
	now := time.Now()
	r := &BlockExecutionResult{
		BlockID:    1,
		Position:   0,
		ExecutedAt: now,
		Duration:   time.Second,
		Stdout:     []string{"output1", "output2"},
	}
	result := r.String()
	if result == "" {
		t.Fatal("expected non-empty string")
	}
	if !Contains(result, "STDOUT") {
		t.Fatalf("expected STDOUT in result: %s", result)
	}
}

func Contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
