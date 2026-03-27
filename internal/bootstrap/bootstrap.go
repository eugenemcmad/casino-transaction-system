package bootstrap

import (
	"casino-transaction-system/internal/app"
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/repository"
	"casino-transaction-system/internal/service"
	httptransport "casino-transaction-system/internal/transport/http"
	kafkatransport "casino-transaction-system/internal/transport/kafka"
	"fmt"
	"net/http"
	"time"
)

func NewApiApp(cfg *config.Config) (*app.ApiApp, error) {
	poolCfg := repository.PoolConfig{
		MaxOpenConns:    cfg.Postgres.PoolMaxOpen,
		MaxIdleConns:    cfg.Postgres.PoolMaxIdle,
		ConnMaxLifetime: time.Duration(cfg.Postgres.ConnMaxLifetimeMinutes) * time.Minute,
	}

	repo, err := repository.NewPostgresRepoWithPool(cfg.Postgres.URL, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres repository: %w", err)
	}

	svc := service.NewTransactionService(repo)
	handler := httptransport.NewTransactionHandler(svc)
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

func NewProcessorApp(cfg *config.Config) (*app.ProcessorApp, error) {
	poolCfg := repository.PoolConfig{
		MaxOpenConns:    cfg.Postgres.PoolMaxOpen,
		MaxIdleConns:    cfg.Postgres.PoolMaxIdle,
		ConnMaxLifetime: time.Duration(cfg.Postgres.ConnMaxLifetimeMinutes) * time.Minute,
	}

	repo, err := repository.NewPostgresRepoWithPool(cfg.Postgres.URL, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("initialize postgres repository: %w", err)
	}

	svc := service.NewTransactionService(repo)
	consumer := kafkatransport.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, svc)

	return app.NewProcessorApp(cfg, consumer, repo), nil
}
