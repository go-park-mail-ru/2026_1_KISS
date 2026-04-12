package ctxutil_test

import (
	"context"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/ctxutil"
)

func TestSetAndGetRequestID(t *testing.T) {
	ctx := ctxutil.SetRequestID(context.Background(), "test-123")
	got := ctxutil.RequestIDFromContext(ctx)
	if got != "test-123" {
		t.Errorf("want test-123, got %s", got)
	}
}

func TestRequestIDFromContext_Empty(t *testing.T) {
	got := ctxutil.RequestIDFromContext(context.Background())
	if got != "" {
		t.Errorf("want empty, got %s", got)
	}
}
