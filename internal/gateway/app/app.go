package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	gwmw "github.com/go-park-mail-ru/2026_1_KISS/internal/gateway/middleware"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/gateway/handler"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/config"
	_ "github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/metrics"
	pbauth "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/auth"
	pbissue "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/issue"
	pbnotebook "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notebook"
	pbnotification "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/notification"
	pbpayment "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/payment"
	pbrunner "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/runner"
	pbstorage "github.com/go-park-mail-ru/2026_1_KISS/pkg/api/storage"
)

type App struct {
	srv         *http.Server
	authConn    *grpc.ClientConn
	nbConn      *grpc.ClientConn
	runConn     *grpc.ClientConn
	storConn    *grpc.ClientConn
	issueConn   *grpc.ClientConn
	notifConn   *grpc.ClientConn
	paymentConn *grpc.ClientConn
	cancelMw    context.CancelFunc
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
	storConn, err := grpc.NewClient(cfg.GRPC.StorageAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		authConn.Close()
		nbConn.Close()
		runConn.Close()
		return nil, fmt.Errorf("dial storage: %w", err)
	}
	issueConn, err := grpc.NewClient(cfg.GRPC.IssueAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		authConn.Close()
		nbConn.Close()
		runConn.Close()
		storConn.Close()
		return nil, fmt.Errorf("dial issue: %w", err)
	}
	notifConn, err := grpc.NewClient(cfg.GRPC.NotificationAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		authConn.Close()
		nbConn.Close()
		runConn.Close()
		storConn.Close()
		issueConn.Close()
		return nil, fmt.Errorf("dial notification: %w", err)
	}
	paymentConn, err := grpc.NewClient(cfg.GRPC.PaymentAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		authConn.Close()
		nbConn.Close()
		runConn.Close()
		storConn.Close()
		issueConn.Close()
		notifConn.Close()
		return nil, fmt.Errorf("dial payment: %w", err)
	}
	authClient := pbauth.NewAuthServiceClient(authConn)
	nbClient := pbnotebook.NewNotebookServiceClient(nbConn)
	runClient := pbrunner.NewRunnerServiceClient(runConn)
	storageClient := pbstorage.NewStorageServiceClient(storConn)
	issueClient := pbissue.NewIssueServiceClient(issueConn)
	notifClient := pbnotification.NewNotificationServiceClient(notifConn)
	paymentClient := pbpayment.NewPaymentServiceClient(paymentConn)

	authHandler := handler.NewAuthHandler(authClient, cfg.Auth.CookieSecure, cfg.Mail.AppURL)
	oauthHandler := handler.NewOAuthHandler(authClient, cfg.Auth.CookieSecure, cfg.OAuth.FrontendURL)
	profileHandler := handler.NewProfileHandler(authClient, cfg.Upload.MaxSize)
	notebookHandler := handler.NewNotebookHandler(nbClient, authClient)
	runnerHandler := handler.NewRunnerHandler(runClient)
	fileHandler := handler.NewFileHandler(storageClient, cfg.Upload.MaxSize)
	healthHandler := handler.NewHealthHandler()
	eventHandler := handler.NewEventHandler(authClient)
	adminHandler := handler.NewAdminHandler(authClient, nbClient, storageClient, notifClient)
	wsHandler := handler.NewWSHandler(authClient, nbClient, runClient)
	statsHandler := handler.NewStatsHandler(authClient, nbClient, storageClient)
	issueHandler := handler.NewIssueHandler(issueClient, authClient)
	paymentHandler := handler.NewPaymentHandler(paymentClient, cfg.YooKassa.WebhookCIDR)

	mux := http.NewServeMux()
	authMw := gwmw.Auth(authClient)
	adminMw := gwmw.AdminOnly()

	authHandler.RegisterRoutes(mux)
	oauthHandler.RegisterRoutes(mux)
	notebookHandler.RegisterRoutes(mux, authMw)
	runnerHandler.RegisterRoutes(mux, authMw)
	profileHandler.RegisterRoutes(mux, authMw)
	fileHandler.RegisterRoutes(mux, authMw)
	healthHandler.RegisterRoutes(mux)
	eventHandler.RegisterRoutes(mux, authMw)
	adminHandler.RegisterRoutes(mux, authMw, adminMw)
	statsHandler.RegisterRoutes(mux, authMw)
	wsHandler.RegisterRoutes(mux)
	issueHandler.RegisterRoutes(mux, authMw, adminMw)
	paymentHandler.RegisterRoutes(mux, authMw)

	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.Upload.Dir))))
	mux.Handle("GET /metrics", promhttp.Handler())

	mwCtx, cancelMw := context.WithCancel(context.Background())

	mws := []middleware.Middleware{
		middleware.Metrics(),
		middleware.RequestID(),
		middleware.Logging(),
		middleware.Recovery(),
		middleware.CORS(cfg.CORS.AllowedOrigins),
		middleware.SecurityHeaders(),
	}
	if cfg.DisableCSRF {
		slog.Warn("CSRF protection is DISABLED (DISABLE_KISS_CSRF=true)")
	} else {
		csrfSkip := map[string]bool{
			"/api/v1/auth/login":       true,
			"/api/v1/auth/register":    true,
			"/api/v1/payments/webhook": true,
		}
		mws = append(mws, middleware.CSRF(csrfSkip))
	}
	mws = append(mws, middleware.RateLimit(mwCtx, cfg.RateLimit.MaxRequests, cfg.RateLimit.Window))

	mwHandler := middleware.Chain(mux, mws...)

	srv := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      mwHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	return &App{
		srv:         srv,
		authConn:    authConn,
		nbConn:      nbConn,
		runConn:     runConn,
		storConn:    storConn,
		issueConn:   issueConn,
		notifConn:   notifConn,
		paymentConn: paymentConn,
		cancelMw:    cancelMw,
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
	a.storConn.Close()
	a.issueConn.Close()
	a.notifConn.Close()
	if a.paymentConn != nil {
		a.paymentConn.Close()
	}
}
