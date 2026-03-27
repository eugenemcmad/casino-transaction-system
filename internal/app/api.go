package app

import (
	"casino-transaction-system/internal/config"
	"context"
	"log/slog"
	"net/http"
)

type resourceCloser interface {
	Close()
}

type ApiApp struct {
	cfg    *config.Config
	server *http.Server
	closer resourceCloser
}

func NewApiApp(cfg *config.Config, server *http.Server, closer resourceCloser) *ApiApp {
	slog.Info(MsgAPIInitialized)

	return &ApiApp{
		cfg:    cfg,
		server: server,
		closer: closer,
	}
}

func (a *ApiApp) Run(ctx context.Context) error {
	slog.Info(MsgStartingAPI, "port", a.cfg.HTTP.Port)

	serverErrCh := make(chan error, 1)
	go func() {
		if err := a.server.ListenAndServe(); err != nil {
			serverErrCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErrCh:
		if err != nil && err != http.ErrServerClosed {
			slog.Error(MsgHTTPServerError, "err", err)
			if a.closer != nil {
				slog.Info(MsgClosingDBConnection)
				a.closer.Close()
			}
			return err
		}
	}

	slog.Info(MsgShuttingDownAPI)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		slog.Error(MsgHTTPServerShutdown, "err", err)
	}

	if a.closer != nil {
		slog.Info(MsgClosingDBConnection)
		a.closer.Close()
	}

	return nil
}
