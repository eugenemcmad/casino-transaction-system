package app

import (
	"casino-transaction-system/internal/config"
	"context"
	"log/slog"
)

type consumerRunner interface {
	Start(ctx context.Context) error
}

// ProcessorApp runs the Kafka consumer and closes shared resources after Stop.
type ProcessorApp struct {
	cfg      *config.Config
	consumer consumerRunner
	closer   resourceCloser
}

// NewProcessorApp constructs the Kafka processor runtime.
func NewProcessorApp(cfg *config.Config, consumer consumerRunner, closer resourceCloser) *ProcessorApp {
	slog.Info(MsgProcessorInitialized)

	return &ProcessorApp{
		cfg:      cfg,
		consumer: consumer,
		closer:   closer,
	}
}

// Run blocks until the consumer returns (typically on ctx cancellation), then closes resources.
func (a *ProcessorApp) Run(ctx context.Context) error {
	slog.Info(MsgStartingProcessor, "topic", a.cfg.Kafka.Topic)

	err := a.consumer.Start(ctx)

	if a.closer != nil {
		slog.Info(MsgClosingDBConnection)
		a.closer.Close()
	}

	if err != nil {
		slog.Error(MsgKafkaConsumerError, "error", err)
		return err
	}

	return nil
}
