package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	gwmw "github.com/go-park-mail-ru/2026_1_KISS/internal/gateway/middleware"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/gateway/handler"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pbnotebook "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
	pbrunner "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/runner"
)

type App struct {
	srv      *http.Server
	authConn *grpc.ClientConn
	nbConn   *grpc.ClientConn
	runConn  *grpc.ClientConn
	cancelMw context.CancelFunc
}

func New(cfg *config.Config) (*App, error) {
	authConn, err := grpc.NewClient(cfg.GRPC.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial auth: %w", err)
	}
	nbConn, err := grpc.NewClient(cfg.GRPC.NotebookAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		authConn.Close()
		return nil, fmt.Errorf("dial notebook: %w", err)
	}
	runConn, err := grpc.NewClient(cfg.GRPC.RunnerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		authConn.Close()
		nbConn.Close()
		return nil, fmt.Errorf("dial runner: %w", err)
	}

	authClient := pbauth.NewAuthServiceClient(authConn)
	nbClient := pbnotebook.NewNotebookServiceClient(nbConn)
	runClient := pbrunner.NewRunnerServiceClient(runConn)

	authHandler := handler.NewAuthHandler(authClient, cfg.Auth.CookieSecure)
	profileHandler := handler.NewProfileHandler(authClient, cfg.Upload.MaxSize)
	notebookHandler := handler.NewNotebookHandler(nbClient)
	runnerHandler := handler.NewRunnerHandler(runClient)
	healthHandler := handler.NewHealthHandler()

	mux := http.NewServeMux()
	authMw := gwmw.Auth(authClient)

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

	mwCtx, cancelMw := context.WithCancel(context.Background())
	mwHandler := middleware.Chain(mux,
		middleware.RequestID(),
		middleware.Logging(),
		middleware.Recovery(),
		middleware.CORS(cfg.CORS.AllowedOrigins),
		middleware.SecurityHeaders(),
		middleware.CSRF(csrfSkip),
		middleware.RateLimit(mwCtx, cfg.RateLimit.MaxRequests, cfg.RateLimit.Window),
	)

	srv := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      mwHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	return &App{
		srv:      srv,
		authConn: authConn,
		nbConn:   nbConn,
		runConn:  runConn,
		cancelMw: cancelMw,
	}, nil
}

func (a *App) Run() error {
	slog.Info("gateway started", "addr", a.srv.Addr)
	return a.srv.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) {
	a.cancelMw()
	if err := a.srv.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	a.authConn.Close()
	a.nbConn.Close()
	a.runConn.Close()
}
