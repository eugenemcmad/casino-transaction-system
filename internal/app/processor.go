package app

import (
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/repository"
	"casino-transaction-system/internal/service"
	transport "casino-transaction-system/internal/transport/kafka"
	"context"
	"log/slog"
	"sync"
)

type ProcessorApp struct {
	cfg      *config.Config
	consumer *transport.Consumer
	repo     *repository.PostgresRepo
	wg       sync.WaitGroup
}

func NewProcessorApp(cfg *config.Config) (*ProcessorApp, error) {
	repo := repository.NewPostgresRepo(cfg.Postgres.URL)
	svc := service.NewTransactionService(repo)
	consumer := transport.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, svc)

	slog.Info(MsgProcessorInitialized)

	return &ProcessorApp{
		cfg:      cfg,
		consumer: consumer,
		repo:     repo,
	}, nil
}

func (a *ProcessorApp) Run(ctx context.Context) error {
	slog.Info(MsgStartingProcessor, "topic", a.cfg.Kafka.Topic)

	err := a.consumer.Start(ctx)
	
	if a.repo != nil {
		slog.Info(MsgClosingDBConnection)
		a.repo.Close()
	}

	if err != nil {
		slog.Error(MsgKafkaConsumerError, "error", err)
		return err
	}

	return nil
}
