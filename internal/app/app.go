package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
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
	cancelReaper  context.CancelFunc
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
	runnerServ := runner_service.NewRunnerService(runnerManager, execSessionRepo, notebookRepo, blockRepo, cfg.Runner.IdleTimeout)

	reaperCtx, cancelReaper := context.WithCancel(context.Background())
	go runnerServ.StartIdleReaper(reaperCtx)

	runnerHandler := runnerhandler.NewRunnerHandler(runnerServ)

	mux := http.NewServeMux()
	authMw := middleware.Auth(authUC)

	authHandler.RegisterRoutes(mux)
	notebookHandler.RegisterRoutes(mux, authMw)
	runnerHandler.RegisterRoutes(mux, authMw)
	profileHandler.RegisterRoutes(mux, authMw)
	healthHandler.RegisterRoutes(mux)

	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.Upload.Dir))))

	csrfSkip := map[string]bool{
		"/api/v1/auth/login":    true,
		"/api/v1/auth/register": true,
	}

	handler := middleware.Chain(mux,
		middleware.RequestID(),
		middleware.Logging(),
		middleware.Recovery(),
		middleware.CORS(cfg.CORS.AllowedOrigins),
		middleware.SecurityHeaders(),
		middleware.CSRF(csrfSkip),
		middleware.RateLimit(300, time.Minute),
	)

	srv := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 180 * time.Second,
	}

	return &App{
		srv:           srv,
		db:            db,
		rdb:           rdb,
		runnerManager: runnerManager,
		cancelReaper:  cancelReaper,
	}, nil
}

func (a *App) Run() error {
	slog.Info("server started", "addr", a.srv.Addr)
	return a.srv.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) {
	a.cancelReaper()
	if err := a.srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	if err := a.db.Close(); err != nil {
		slog.Error("db close error", "error", err)
	}
	if a.rdb != nil {
		if err := a.rdb.Close(); err != nil {
			slog.Error("redis close error", "error", err)
		}
	}
	if a.runnerManager != nil {
		if err := a.runnerManager.Close(); err != nil {
			slog.Error("runner manager close error", "error", err)
		}
	}
}
