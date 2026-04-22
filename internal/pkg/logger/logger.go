package logger

import (
	"context"
	"log/slog"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/ctxutil"
)

func Info(ctx context.Context, msg string, args ...any) {
	slog.InfoContext(ctx, msg, withRequestID(ctx, args)...)
}

func Error(ctx context.Context, msg string, args ...any) {
	slog.ErrorContext(ctx, msg, withRequestID(ctx, args)...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	slog.WarnContext(ctx, msg, withRequestID(ctx, args)...)
}

func withRequestID(ctx context.Context, args []any) []any {
	if id := ctxutil.RequestIDFromContext(ctx); id != "" {
		args = append(args, "request_id", id)
	}
	return args
}
