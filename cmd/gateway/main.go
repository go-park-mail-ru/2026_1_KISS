package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/gateway/app"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
)

func main() {
	cfg := config.Load()

	a, err := app.New(cfg)
	if err != nil {
		slog.Error("failed to create gateway", "error", err)
		os.Exit(1)
	}

	go func() {
		if err := a.Run(); err != nil {
			slog.Error("gateway error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gateway")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a.Shutdown(ctx)
}
