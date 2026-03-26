package app

import (
	"casino-transaction-system/internal/config"
	"context"
	"log/slog"
)

// consumerRunner is a local app-layer contract for starting message consumption.
// It keeps ProcessorApp independent from a specific transport package (e.g. Kafka),
// so runtime wiring stays in bootstrap and app logic depends only on behavior.
// In tests, this seam allows cheap fakes/stubs to verify shutdown/error flows
// without bringing up Kafka or transport-level infrastructure.
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
