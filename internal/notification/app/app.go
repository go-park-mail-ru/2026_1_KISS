package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	notificationgrpc "github.com/go-park-mail-ru/2026_1_KISS/internal/notification/grpc"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/metrics"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notification"
)

type App struct {
	grpcServer *grpc.Server
	listener   net.Listener
}

func New(cfg *config.Config, grpcPort string) (*App, error) {
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	metrics.StartMetricsServer(":" + cfg.Metrics.Port)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcutil.RecoveryUnaryInterceptor(),
			grpcutil.MetricsUnaryInterceptor("notification"),
			grpcutil.LoggingUnaryInterceptor(),
		),
	)

	server := notificationgrpc.NewServer(
		cfg.Mail.From,
		cfg.Mail.SMTPHost,
		cfg.Mail.SMTPPort,
	)
	pb.RegisterNotificationServiceServer(srv, server)

	return &App{
		grpcServer: srv,
		listener:   lis,
	}, nil
}

func (a *App) Run() error {
	slog.Info("notification service started", "addr", a.listener.Addr())
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Shutdown(_ context.Context) {
	a.grpcServer.GracefulStop()
}
