package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"google.golang.org/grpc"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/domain"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/filestorage"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/metrics"
	storagegrpc "github.com/go-park-mail-ru/2026_1_KISS/internal/storage/grpc"
	storagepg "github.com/go-park-mail-ru/2026_1_KISS/internal/storage/repository/postgres"
	storageusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/storage/usecase"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
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

	fileRepo := storagepg.NewFileRepository(db)
	fs := filestorage.NewLocalStorage(cfg.Upload.Dir, "/uploads/")

	maxSizes := map[domain.FileCategory]int64{
		domain.FileCategoryAvatar:   cfg.Upload.MaxAvatarSize,
		domain.FileCategoryDataset:  cfg.Upload.MaxDatasetSize,
		domain.FileCategoryFeedback: cfg.Upload.MaxFeedbackSize,
		domain.FileCategoryGeneral:  cfg.Upload.MaxSize,
	}

	storageUC := storageusecase.New(fileRepo, fs, maxSizes)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	metricsSrv := metrics.StartMetricsServer(":" + cfg.Metrics.Port)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcutil.RecoveryUnaryInterceptor(),
			grpcutil.MetricsUnaryInterceptor("storage"),
			grpcutil.LoggingUnaryInterceptor(),
		),
	)
	pb.RegisterStorageServiceServer(srv, storagegrpc.NewServer(storageUC))

	return &App{
		grpcServer: srv,
		listener:   lis,
		db:         db,
		metricsSrv: metricsSrv,
	}, nil
}

func (a *App) Run() error {
	slog.Info("storage service started", "addr", a.listener.Addr().String())
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Shutdown(_ context.Context) {
	metrics.ShutdownMetricsServer(a.metricsSrv)
	a.grpcServer.GracefulStop()
	if err := a.db.Close(); err != nil {
		slog.Error("db close error", "error", err)
	}
}
