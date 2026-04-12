package logger_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/ctxutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/logger"
)

func TestInfoAppendsRequestID(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))

	ctx := ctxutil.SetRequestID(context.Background(), "req-abc")
	logger.Info(ctx, "test message", "key", "val")

	output := buf.String()
	if !strings.Contains(output, "request_id=req-abc") {
		t.Errorf("expected request_id in log, got: %s", output)
	}
	if !strings.Contains(output, "key=val") {
		t.Errorf("expected key=val in log, got: %s", output)
	}
}

func TestInfoWithoutRequestID(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))

	logger.Info(context.Background(), "no id")

	output := buf.String()
	if strings.Contains(output, "request_id") {
		t.Errorf("unexpected request_id in log: %s", output)
	}
}

func TestErrorAppendsRequestID(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	slog.SetDefault(slog.New(handler))

	ctx := ctxutil.SetRequestID(context.Background(), "req-err")
	logger.Error(ctx, "err message")

	output := buf.String()
	if !strings.Contains(output, "request_id=req-err") {
		t.Errorf("expected request_id in error log, got: %s", output)
	}
}
