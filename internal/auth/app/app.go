package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	authgrpc "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/grpc"
	authpg "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository/postgres"
	authredis "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository/redis"
	authusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/filestorage"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/grpcutil"
	profileusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/profile/usecase"
	pb "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
)

type App struct {
	grpcServer *grpc.Server
	listener   net.Listener
	db         *sql.DB
	rdb        *redisv9.Client
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

	authUC := authusecase.New(userRepo, sessionRepo, cfg.Auth.SessionTTL)

	fs := filestorage.NewLocalStorage(cfg.Upload.Dir, "/uploads/")
	profileUC := profileusecase.New(userRepo, fs, cfg.Upload.MaxSize)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcutil.RecoveryUnaryInterceptor(),
			grpcutil.LoggingUnaryInterceptor(),
		),
	)
	pb.RegisterAuthServiceServer(srv, authgrpc.NewServer(authUC, profileUC))

	return &App{
		grpcServer: srv,
		listener:   lis,
		db:         db,
		rdb:        rdb,
	}, nil
}

func (a *App) Run() error {
	slog.Info("auth service started", "addr", a.listener.Addr().String())
	return a.grpcServer.Serve(a.listener)
}

func (a *App) Shutdown(_ context.Context) {
	a.grpcServer.GracefulStop()
	if err := a.db.Close(); err != nil {
		slog.Error("db close error", "error", err)
	}
	if a.rdb != nil {
		if err := a.rdb.Close(); err != nil {
			slog.Error("redis close error", "error", err)
		}
	}
}
