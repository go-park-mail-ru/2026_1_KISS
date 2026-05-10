package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/payment/app"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
)

func main() {
	cfg := config.Load()

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9007"
	}

	a, err := app.New(cfg, grpcPort)
	if err != nil {
		slog.Error("failed to create payment app", "error", err)
		os.Exit(1)
	}

	go func() {
		if err := a.Run(); err != nil {
			slog.Error("payment service error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down payment service")
	a.Shutdown(context.Background())
}
