package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authgrpc "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/grpc"
	authpg "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository/postgres"
	authredis "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository/redis"
	authusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/metrics"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pbnotification "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notification"
	pbstorage "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

type App struct {
	grpcServer    *grpc.Server
	listener      net.Listener
	db            *sql.DB
	rdb           *redisv9.Client
	storConn      *grpc.ClientConn
	notifConn     *grpc.ClientConn
	metricsSrv    *http.Server
	cancelCleanup context.CancelFunc
}

func New(cfg *config.Config, grpcPort string) (*App, error) {
	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	rdb := redisv9.NewClient(&redisv9.Options{
		Addr:         fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	userRepo := authpg.NewUserRepository(db)
	sessionRepo := authredis.NewSessionRepository(rdb)
	verificationRepo := authpg.NewVerificationRepository(db)
	eventRepo := authpg.NewEventRepository(db)
	subViewRepo := authpg.NewSubscriptionViewRepository(db)

	notifConn, err := grpc.NewClient(cfg.GRPC.NotificationAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial notification: %w", err)
	}
	notifClient := pbnotification.NewNotificationServiceClient(notifConn)
	notifAdapter := authgrpc.NewNotificationAdapter(notifClient, cfg.Mail.AppURL)

	authUC := authusecase.New(
		userRepo,
		sessionRepo,
		verificationRepo,
		notifAdapter,
		cfg.Auth.SessionTTL,
	).WithSubscriptionView(subViewRepo)

	storConn, err := grpc.NewClient(cfg.GRPC.StorageAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial storage: %w", err)
	}
	storageClient := pbstorage.NewStorageServiceClient(storConn)
	uploader := authusecase.NewStorageUploader(storageClient)
	profileUC := authusecase.NewProfileUsecase(userRepo, uploader, cfg.Upload.MaxAvatarSize)
	eventUC := authusecase.NewEventUsecase(eventRepo, userRepo, subViewRepo)
	adminUC := authusecase.NewAdminUsecase(userRepo, eventRepo)
	statsUC := authusecase.NewStatsUsecase(userRepo, eventRepo)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	metricsSrv := metrics.StartMetricsServer(":" + cfg.Metrics.Port)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcutil.RecoveryUnaryInterceptor(),
			grpcutil.MetricsUnaryInterceptor("auth"),
			grpcutil.LoggingUnaryInterceptor(),
		),
	)
	pb.RegisterAuthServiceServer(srv, authgrpc.NewServer(authUC, profileUC, eventUC, adminUC, statsUC))

	cleanupCtx, cancelCleanup := context.WithCancel(context.Background())
	go authUC.StartCleanupLoop(cleanupCtx)

	return &App{
		grpcServer:    srv,
		listener:      lis,
		db:            db,
		rdb:           rdb,
		storConn:      storConn,
		notifConn:     notifConn,
		metricsSrv:    metricsSrv,
		cancelCleanup: cancelCleanup,
	}, nil
}

func (a *App) Run() error {
	slog.Info("auth service started", "addr", a.listener.Addr().String())
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Shutdown(_ context.Context) {
	a.cancelCleanup()
	metrics.ShutdownMetricsServer(a.metricsSrv)
	a.grpcServer.GracefulStop()
	if err := a.db.Close(); err != nil {
		slog.Error("db close error", "error", err)
	}
	if a.rdb != nil {
		if err := a.rdb.Close(); err != nil {
			slog.Error("redis close error", "error", err)
		}
	}
	if a.storConn != nil {
		a.storConn.Close()
	}
	if a.notifConn != nil {
		a.notifConn.Close()
	}
}
