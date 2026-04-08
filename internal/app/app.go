package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	redisv9 "github.com/redis/go-redis/v9"

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
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/filestorage"
	profilehttp "github.com/go-park-mail-ru/2026_1_KISS/internal/profile/delivery/http"
	profileusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/profile/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/container"
	runnerhandler "github.com/go-park-mail-ru/2026_1_KISS/internal/runner/delivery"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/runner_service"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/runner/session_repository"
)

type App struct {
	srv *http.Server
	db  *sql.DB
	rdb *redisv9.Client

	runnerManager container.Manager
}

func NoOpMiddleware(next http.Handler) http.Handler {
	return next
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

	fs := filestorage.NewLocalStorage(cfg.Upload.Dir, "/uploads/")
	profileUC := profileusecase.New(userRepo, fs, cfg.Upload.MaxSize)

	authHandler := authhttp.New(authUC)
	notebookHandler := nbhttp.New(notebookUC)
	profileHandler := profilehttp.New(profileUC, cfg.Upload.MaxSize)
	healthHandler := health.New(db)

	runnerManager, err := container.NewManager(cfg.Runner)
	if err != nil {
		_ = rdb.Close()
		_ = db.Close()
		return nil, fmt.Errorf("init runner manager: %w", err)
	}
	execSessionRepo := session_repository.NewExecutionSessionRepository()
	runnerServ := runner_service.NewRunnerService(runnerManager, execSessionRepo, notebookRepo, blockRepo)
	runnerHandler := runnerhandler.NewRunnerHandler(runnerServ)

	mux := http.NewServeMux()
	authMw := middleware.Auth(authUC)

	authHandler.RegisterRoutes(mux)
	notebookHandler.RegisterRoutes(mux, authMw)
	runnerHandler.RegisterRoutes(mux, NoOpMiddleware)
	profileHandler.RegisterRoutes(mux, authMw)
	healthHandler.RegisterRoutes(mux)

	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.Upload.Dir))))

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
