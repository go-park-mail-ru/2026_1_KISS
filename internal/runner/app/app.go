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
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/runner_service"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/session_repository"
	pbnotebook "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
	pbrunner "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/runner"
)

type App struct {
	grpcServer    *grpc.Server
	listener      net.Listener
	runnerManager container.Manager
	cancelReaper  context.CancelFunc
	nbConn        *grpc.ClientConn
	metricsSrv    *http.Server
}

func New(cfg *config.Config, grpcPort string) (*App, error) {
	nbConn, err := grpc.NewClient(cfg.GRPC.NotebookAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connect notebook service: %w", err)
	}
	nbClient := pbnotebook.NewNotebookServiceClient(nbConn)

	notebookAdapter := runnergrpc.NewNotebookAdapter(nbClient)
	blockAdapter := runnergrpc.NewBlockAdapter(nbClient)

	runnerManager, err := container.NewManager(cfg.Runner)
	if err != nil {
		nbConn.Close()
		return nil, fmt.Errorf("init runner manager: %w", err)
	}

	execSessionRepo := session_repository.NewExecutionSessionRepository(cfg.Runner.ExecutionTimeout)
	runnerSvc := runner_service.NewRunnerService(runnerManager, execSessionRepo, notebookAdapter, blockAdapter, cfg.Runner.IdleTimeout)

	reaperCtx, cancelReaper := context.WithCancel(context.Background())
	go runnerSvc.StartIdleReaper(reaperCtx)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		cancelReaper()
		nbConn.Close()
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
		grpcServer:    srv,
		listener:      lis,
		runnerManager: runnerManager,
		cancelReaper:  cancelReaper,
		nbConn:        nbConn,
		metricsSrv:    metricsSrv,
	}, nil
}

func (a *App) Run() error {
	slog.Info("runner service started", "addr", a.listener.Addr().String())
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Shutdown(_ context.Context) {
	metrics.ShutdownMetricsServer(a.metricsSrv)
	a.cancelReaper()
	a.grpcServer.GracefulStop()
	if a.runnerManager != nil {
		if err := a.runnerManager.Close(); err != nil {
			slog.Error("runner manager close error", "error", err)
		}
	}
	if a.nbConn != nil {
		a.nbConn.Close()
	}
}
