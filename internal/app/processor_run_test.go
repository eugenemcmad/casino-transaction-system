package app

import (
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/service"
	transport "casino-transaction-system/internal/transport/kafka"
	"context"
	"testing"
)

func TestProcessorApp_Run_CanceledContext(t *testing.T) {
	cfg := &config.Config{}
	cfg.Kafka.Brokers = []string{"127.0.0.1:9092"}
	cfg.Kafka.Topic = "test-topic"
	cfg.Kafka.GroupID = "test-group"

	consumer := transport.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID, service.NewTransactionService(nil), cfg.Kafka)
	app := &ProcessorApp{
		cfg:      cfg,
		consumer: consumer,
		closer:   noopCloser{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := app.Run(ctx); err != nil {
		t.Fatalf("Run() with canceled context returned error: %v", err)
	}
}
