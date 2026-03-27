// Package app hosts runnable application shells (API HTTP server and Kafka processor).
package app

import (
	"casino-transaction-system/internal/config"
	"context"
	"log/slog"
	"net/http"
	"sync"
)

type resourceCloser interface {
	Close()
}

// ApiApp runs the HTTP server and owns lifecycle of injected resources (e.g. database).
type ApiApp struct {
	cfg       *config.Config
	server    *http.Server
	closer    resourceCloser
	closeOnce sync.Once
}

// NewApiApp constructs the HTTP API runtime. closer is closed on shutdown or fatal listen errors.
func NewApiApp(cfg *config.Config, server *http.Server, closer resourceCloser) *ApiApp {
	slog.Info(MsgAPIInitialized)

	return &ApiApp{
		cfg:    cfg,
		server: server,
		closer: closer,
	}
}

// Run starts ListenAndServe in a goroutine, waits for ctx cancel or a non-recoverable listen error,
// then performs graceful Shutdown and closes resources once.
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
			a.closeResources()
			return err
		}
	}

	slog.Info(MsgShuttingDownAPI)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		slog.Error(MsgHTTPServerShutdown, "err", err)
	}

	a.closeResources()

	return nil
}

// closeResources invokes closer at most once (e.g. DB pool).
func (a *ApiApp) closeResources() {
	a.closeOnce.Do(func() {
		if a.closer != nil {
			slog.Info(MsgClosingDBConnection)
			a.closer.Close()
		}
	})
}
