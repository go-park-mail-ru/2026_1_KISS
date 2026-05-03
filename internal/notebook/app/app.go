package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"google.golang.org/grpc"

	nbgrpc "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/grpc"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/hub"
	nbpg "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository/postgres"
	nbusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/metrics"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
)

type App struct {
	grpcServer *grpc.Server
	listener   net.Listener
	db         *sql.DB
	metricsSrv *http.Server
}

func New(cfg *config.Config, grpcPort string) (*App, error) {
	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	notebookRepo := nbpg.NewNotebookRepository(db)
	blockRepo := nbpg.NewBlockRepository(db)
	permRepo := nbpg.NewPermissionRepository(db)
	commentRepo := nbpg.NewCommentRepository(db)
	eventHub := hub.New()
	notebookUC := nbusecase.New(notebookRepo, blockRepo, permRepo, commentRepo, eventHub)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	metricsSrv := metrics.StartMetricsServer(":" + cfg.Metrics.Port)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcutil.RecoveryUnaryInterceptor(),
			grpcutil.MetricsUnaryInterceptor("notebook"),
			grpcutil.LoggingUnaryInterceptor(),
		),
	)
	pb.RegisterNotebookServiceServer(srv, nbgrpc.NewServer(notebookUC, blockRepo, eventHub))

	return &App{
		grpcServer: srv,
		listener:   lis,
		db:         db,
		metricsSrv: metricsSrv,
	}, nil
}

func (a *App) Run() error {
	slog.Info("notebook service started", "addr", a.listener.Addr().String())
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Shutdown(_ context.Context) {
	metrics.ShutdownMetricsServer(a.metricsSrv)
	a.grpcServer.GracefulStop()
	if err := a.db.Close(); err != nil {
		slog.Error("db close error", "error", err)
	}
}
