package app

import (
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/repository"
	"casino-transaction-system/internal/service"
	transport "casino-transaction-system/internal/transport/http"
	"context"
	"log/slog"
	"net/http"
)

type ApiApp struct {
	cfg    *config.Config
	server *http.Server
	repo   *repository.PostgresRepo 
}

func NewApiApp(cfg *config.Config) *ApiApp {
	repo := repository.NewPostgresRepo(cfg.Postgres.URL)
	svc := service.NewTransactionService(repo)

	handler := transport.NewTransactionHandler(svc)
	router := transport.NewRouter(handler)

	server := &http.Server{
		Addr:    ":" + cfg.HTTP.Port,
		Handler: router,
	}

	slog.Info(MsgAPIInitialized)

	return &ApiApp{
		cfg:    cfg,
		server: server,
		repo:   repo,
	}
}

func (a *ApiApp) Run(ctx context.Context) error {
	slog.Info(MsgStartingAPI, "port", a.cfg.HTTP.Port)

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error(MsgHTTPServerError, "err", err)
		}
	}()

	<-ctx.Done()

	slog.Info(MsgShuttingDownAPI)
	
	shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		slog.Error(MsgHTTPServerShutdown, "err", err)
	}

	if a.repo != nil {
		slog.Info(MsgClosingDBConnection)
		a.repo.Close()
	}

	return nil
}
