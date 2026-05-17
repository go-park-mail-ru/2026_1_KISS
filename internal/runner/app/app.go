package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/metrics"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/container"
	runnergrpc "github.com/go-park-mail-ru/2026_1_KISS/internal/runner/grpc"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/pool"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/runner_service"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/session_repository"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/snapshot"
	pbnotebook "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
	pbrunner "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/runner"
	pbstorage "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

type App struct {
	grpcServer   *grpc.Server
	listener     net.Listener
	workerPool   *pool.Pool
	cancelReaper context.CancelFunc
	nbConn       *grpc.ClientConn
	storageConn  *grpc.ClientConn
	metricsSrv   *http.Server
}

func New(cfg *config.Config, grpcPort string) (*App, error) {
	nbConn, err := grpc.NewClient(cfg.GRPC.NotebookAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connect notebook service: %w", err)
	}
	nbClient := pbnotebook.NewNotebookServiceClient(nbConn)

	storageConn, err := grpc.NewClient(cfg.GRPC.StorageAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		nbConn.Close()
		return nil, fmt.Errorf("connect storage service: %w", err)
	}
	storageClient := pbstorage.NewStorageServiceClient(storageConn)

	notebookAdapter := runnergrpc.NewNotebookAdapter(nbClient)
	blockAdapter := runnergrpc.NewBlockAdapter(nbClient)

	runnerManager, err := container.NewManager(cfg.Runner)
	if err != nil {
		nbConn.Close()
		storageConn.Close()
		return nil, fmt.Errorf("init runner manager: %w", err)
	}

	poolCtx, cancelPool := context.WithCancel(context.Background())
	workerPool, err := pool.New(poolCtx, runnerManager, "python", cfg.Runner.PoolSize, cfg.Runner.QueueMax)
	if err != nil {
		cancelPool()
		nbConn.Close()
		storageConn.Close()
		if closeErr := runnerManager.Close(); closeErr != nil {
			slog.Error("runner manager close error", "error", closeErr)
		}
		return nil, fmt.Errorf("init worker pool: %w", err)
	}

	snapshotRepo := snapshot.NewStorageRepository(storageClient, cfg.Runner.SnapshotMaxBytes)
	execSessionRepo := session_repository.NewExecutionSessionRepository(cfg.Runner.ExecutionTimeout)
	runnerSvc := runner_service.NewRunnerService(
		workerPool, snapshotRepo, execSessionRepo,
		notebookAdapter, blockAdapter, cfg.Runner.IdleTimeout,
	)

	reaperCtx, cancelReaper := context.WithCancel(context.Background())
	go runnerSvc.StartIdleReaper(reaperCtx)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		cancelReaper()
		cancelPool()
		nbConn.Close()
		storageConn.Close()
		return nil, fmt.Errorf("listen: %w", err)
	}

	metricsSrv := metrics.StartMetricsServer(":" + cfg.Metrics.Port)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcutil.RecoveryUnaryInterceptor(),
			grpcutil.MetricsUnaryInterceptor("runner"),
			grpcutil.LoggingUnaryInterceptor(),
		),
	)
	pbrunner.RegisterRunnerServiceServer(srv, runnergrpc.NewServer(runnerSvc, nbClient))

	return &App{
		grpcServer:   srv,
		listener:     lis,
		workerPool:   workerPool,
		cancelReaper: cancelReaper,
		nbConn:       nbConn,
		storageConn:  storageConn,
		metricsSrv:   metricsSrv,
	}, nil
}

func (a *App) Run() error {
	slog.Info("runner service started", "addr", a.listener.Addr().String())
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Shutdown(ctx context.Context) {
	metrics.ShutdownMetricsServer(a.metricsSrv)
	a.cancelReaper()
	a.grpcServer.GracefulStop()
	if a.workerPool != nil {
		a.workerPool.Shutdown(ctx)
	}
	if a.nbConn != nil {
		a.nbConn.Close()
	}
	if a.storageConn != nil {
		a.storageConn.Close()
	}
}
