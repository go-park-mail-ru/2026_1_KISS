package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"google.golang.org/grpc"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/issue/repository/postgres"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/issue/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/metrics"

	issuegrpc "github.com/go-park-mail-ru/2026_1_KISS/internal/issue/grpc"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/issue"
)

type App struct {
	grpcServer *grpc.Server
	listener   net.Listener
	metricsSrv *http.Server
}

func New(cfg *config.Config, grpcPort string) (*App, error) {
	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	issueRepo := postgres.NewIssueRepository(db)
	msgRepo := postgres.NewIssueMessageRepository(db)

	issueUC := usecase.NewIssueService(issueRepo)
	msgUC := usecase.NewIssueMessageService(msgRepo)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	metricsSrv := metrics.StartMetricsServer(":" + cfg.Metrics.Port)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcutil.RecoveryUnaryInterceptor(),
			grpcutil.MetricsUnaryInterceptor("issue"),
			grpcutil.LoggingUnaryInterceptor(),
		),
	)

	pb.RegisterIssueServiceServer(srv, issuegrpc.NewServer(issueUC, msgUC))

	return &App{
		grpcServer: srv,
		listener:   lis,
		metricsSrv: metricsSrv,
	}, nil
}

func (a *App) Run() error {
	slog.Info("issue service started", "addr", a.listener.Addr())
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Shutdown(_ context.Context) {
	metrics.ShutdownMetricsServer(a.metricsSrv)
	a.grpcServer.GracefulStop()
}
