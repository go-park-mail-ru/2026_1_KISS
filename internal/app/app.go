package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	authhttp "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/delivery/http"
	authpg "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository/postgres"
	authredis "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository/redis"
	authusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/health"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	nbhttp "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/delivery/http"
	nbpg "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository/postgres"
	nbusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/container"
	redisv9 "github.com/redis/go-redis/v9"
)

type App struct {
	srv *http.Server
	db  *sql.DB
	rdb *redisv9.Client

	runnerManager runner.Manager
}

func New(cfg *config.Config) (*App, error) {
	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		return nil, err
	}

	rdb := redisv9.NewClient(&redisv9.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       0,
	})

	userRepo := authpg.NewUserRepository(db)
	sessionRepo := authredis.NewSessionRepository(rdb)
	notebookRepo := nbpg.NewNotebookRepository(db)
	blockRepo := nbpg.NewBlockRepository(db)

	authUC := authusecase.New(userRepo, sessionRepo, cfg.Auth.SessionTTL)
	notebookUC := nbusecase.New(notebookRepo, blockRepo)

	authHandler := authhttp.New(authUC)
	notebookHandler := nbhttp.New(notebookUC)
	healthHandler := health.New(db)

	mux := http.NewServeMux()
	authMw := middleware.Auth(authUC)

	authHandler.RegisterRoutes(mux)
	notebookHandler.RegisterRoutes(mux, authMw)
	healthHandler.RegisterRoutes(mux)

	handler := middleware.Chain(mux,
		middleware.RequestID(),
		middleware.Logging(),
		middleware.Recovery(),
		middleware.CORS(cfg.CORS.AllowedOrigins),
	)

	srv := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	runnerManager, err := container.NewManager(cfg.Runner)
	if err != nil {
		_ = rdb.Close()
		_ = db.Close()
		return nil, fmt.Errorf("init runner manager: %w", err)
	}

	return &App{
		srv:           srv,
		db:            db,
		rdb:           rdb,
		runnerManager: runnerManager,
	}, nil
}

func (a *App) Run() error {
	log.Printf("server started on %s", a.srv.Addr)
	return a.srv.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) {
	if err := a.srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	if err := a.db.Close(); err != nil {
		log.Printf("db close error: %v", err)
	}
	if a.rdb != nil {
		if err := a.rdb.Close(); err != nil {
			log.Printf("redis close error: %v", err)
		}
	}
	if a.runnerManager != nil {
		if err := a.runnerManager.Close(); err != nil {
			log.Printf("runner manager close error: %v", err)
		}
	}
}
