package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	paymentgrpc "github.com/go-park-mail-ru/2026_1_KISS/internal/payment/grpc"
	paymentpg "github.com/go-park-mail-ru/2026_1_KISS/internal/payment/repository/postgres"
	paymentusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/payment/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/payment/yookassa"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/metrics"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/payment"
)

type App struct {
	grpcServer *grpc.Server
	listener   net.Listener
	db         *sql.DB
	authConn   *grpc.ClientConn
	metricsSrv *http.Server
}

func New(cfg *config.Config, grpcPort string) (*App, error) {
	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	paymentRepo := paymentpg.NewPaymentRepository(db)
	planRepo := paymentpg.NewPlanRepository(db)
	subRepo := paymentpg.NewSubscriptionRepository(db)

	ykClient := yookassa.NewClient(cfg.YooKassa.ShopID, cfg.YooKassa.SecretKey, cfg.YooKassa.APIBase)

	authConn, err := grpc.NewClient(cfg.GRPC.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("dial auth: %w", err)
	}
	authClient := pbauth.NewAuthServiceClient(authConn)
	authAdapter := paymentusecase.NewGRPCAuthAdapter(authClient)

	uc := paymentusecase.New(paymentRepo, planRepo, subRepo, ykClient, authAdapter, cfg.YooKassa.ReturnURL)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		_ = db.Close()
		authConn.Close()
		return nil, fmt.Errorf("listen: %w", err)
	}

	metricsSrv := metrics.StartMetricsServer(":" + cfg.Metrics.Port)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcutil.RecoveryUnaryInterceptor(),
			grpcutil.MetricsUnaryInterceptor("payment"),
			grpcutil.LoggingUnaryInterceptor(),
		),
	)
	pb.RegisterPaymentServiceServer(srv, paymentgrpc.NewServer(uc))

	return &App{
		grpcServer: srv,
		listener:   lis,
		db:         db,
		authConn:   authConn,
		metricsSrv: metricsSrv,
	}, nil
}

func (a *App) Run() error {
	slog.Info("payment service started", "addr", a.listener.Addr().String())
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Shutdown(_ context.Context) {
	metrics.ShutdownMetricsServer(a.metricsSrv)
	a.grpcServer.GracefulStop()
	if a.authConn != nil {
		a.authConn.Close()
	}
	if err := a.db.Close(); err != nil {
		slog.Error("db close error", "error", err)
	}
}
