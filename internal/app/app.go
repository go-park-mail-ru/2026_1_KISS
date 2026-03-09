package app

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	authhttp "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/delivery/http"
	authpg "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/repository/postgres"
	authusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/auth/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	nbhttp "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/delivery/http"
	nbpg "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/repository/postgres"
	nbusecase "github.com/go-park-mail-ru/2026_1_KISS/internal/notebook/usecase"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/database"
)

type App struct {
	srv *http.Server
	db  *sql.DB
}

func New(cfg *config.Config) (*App, error) {
	db, err := database.Connect(cfg.Database.DSN())
	if err != nil {
		return nil, err
	}

	userRepo := authpg.NewUserRepository(db)
	sessionRepo := authpg.NewSessionRepository(db)
	notebookRepo := nbpg.NewNotebookRepository(db)
	blockRepo := nbpg.NewBlockRepository(db)

	authUC := authusecase.New(userRepo, sessionRepo, cfg.Auth.SessionTTL)
	notebookUC := nbusecase.New(notebookRepo, blockRepo)

	authHandler := authhttp.New(authUC)
	notebookHandler := nbhttp.New(notebookUC)

	mux := http.NewServeMux()
	authMw := middleware.Auth(authUC)

	authHandler.RegisterRoutes(mux)
	notebookHandler.RegisterRoutes(mux, authMw)

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

	return &App{srv: srv, db: db}, nil
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
}
