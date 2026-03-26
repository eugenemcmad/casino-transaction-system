// Package bootstrap is the composition root: wires config, repository, transports, and app shells.
package bootstrap

import (
	"casino-transaction-system/internal/app"
	"casino-transaction-system/internal/config"
	basemetrics "casino-transaction-system/internal/observability/metrics"
	"casino-transaction-system/internal/repository"
	"casino-transaction-system/internal/service"
	httptransport "casino-transaction-system/internal/transport/http"
	kafkatransport "casino-transaction-system/internal/transport/kafka"
	"fmt"
	"net/http"
	"time"
)

// NewApiApp builds Postgres, services, HTTP router, and returns an app.ApiApp ready to Run.
func NewApiApp(cfg *config.Config) (*app.ApiApp, error) {
	metricsSink := basemetrics.NewLogSink()

	poolCfg := repository.PoolConfig{
		MaxOpenConns:    cfg.Postgres.PoolMaxOpen,
		MaxIdleConns:    cfg.Postgres.PoolMaxIdle,
		ConnMaxLifetime: time.Duration(cfg.Postgres.ConnMaxLifetimeMinutes) * time.Minute,
	}

	repo, err := repository.NewPostgresRepoWithPoolAndMetrics(cfg.Postgres.URL, poolCfg, metricsSink)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres repository: %w", err)
	}

	svc := service.NewTransactionService(repo)
	handler := httptransport.NewTransactionHandlerWithMetrics(svc, metricsSink)
	router := httptransport.NewRouter(handler)

	server := &http.Server{
		Addr:              ":" + cfg.HTTP.Port,
		Handler:           router,
		ReadHeaderTimeout: time.Duration(cfg.HTTP.ReadHeaderTimeoutSeconds) * time.Second,
		ReadTimeout:       time.Duration(cfg.HTTP.ReadTimeoutSeconds) * time.Second,
		WriteTimeout:      time.Duration(cfg.HTTP.WriteTimeoutSeconds) * time.Second,
		IdleTimeout:       time.Duration(cfg.HTTP.IdleTimeoutSeconds) * time.Second,
	}

	return app.NewApiApp(cfg, server, repo), nil
}

// NewProcessorApp builds Postgres, services, Kafka consumer, and returns a ProcessorApp ready to Run.
func NewProcessorApp(cfg *config.Config) (*app.ProcessorApp, error) {
	metricsSink := basemetrics.NewLogSink()

	poolCfg := repository.PoolConfig{
		MaxOpenConns:    cfg.Postgres.PoolMaxOpen,
		MaxIdleConns:    cfg.Postgres.PoolMaxIdle,
		ConnMaxLifetime: time.Duration(cfg.Postgres.ConnMaxLifetimeMinutes) * time.Minute,
	}

	repo, err := repository.NewPostgresRepoWithPoolAndMetrics(cfg.Postgres.URL, poolCfg, metricsSink)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres repository: %w", err)
	}

	svc := service.NewTransactionService(repo)
	consumer := kafkatransport.NewConsumerWithMetrics(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, svc, cfg.Kafka, metricsSink)

	return app.NewProcessorApp(cfg, consumer, repo), nil
}
