package kafka

import (
	"casino-transaction-system/internal/boundary"
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/service"
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"time"

	"github.com/segmentio/kafka-go"
)

type messageReader interface {
	ReadMessage(context.Context) (kafka.Message, error)
	Close() error
}

type Consumer struct {
	reader messageReader
	svc    service.TransactionService
	cfg    config.Kafka
}

func NewConsumer(brokers []string, topic, groupID string, svc service.TransactionService, cfg config.Kafka) *Consumer {
	slog.Debug("Initializing Kafka Consumer", "brokers", brokers, "topic", topic, "groupID", groupID)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: DefaultMinBytes,
		MaxBytes: DefaultMaxBytes,
	})

	return &Consumer{
		reader: reader,
		svc:    svc,
		cfg:    withDefaultKafkaConfig(cfg),
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	slog.Info("Starting Kafka consumer loop...")
	c.cfg = withDefaultKafkaConfig(c.cfg)
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info(MsgKafkaShuttingDown)
				return c.reader.Close()
			}
			meta := boundary.Classify(err)
			slog.Error(MsgFailedToReadMessage, "error", err, "error_code", meta.Code)
			c.pauseWithJitter(ctx)
			continue
		}

		slog.Debug(MsgKafkaMessageReceived, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)

		var dto TransactionDTO
		if err := json.Unmarshal(msg.Value, &dto); err != nil {
			slog.Error(MsgFailedToUnmarshalMessage,
				"error", err,
				"raw_payload", string(msg.Value),
				"offset", msg.Offset,
			)
			continue
		}

		t, err := dto.ToDomain()
		if err != nil {
			meta := boundary.Classify(err)
			slog.Warn("Kafka transaction amount parse failed",
				"error", err,
				"error_code", meta.Code,
				"dto", dto,
				"raw_payload", string(msg.Value),
				"offset", msg.Offset,
			)
			continue
		}

		if err := t.Validate(); err != nil {
			meta := boundary.Classify(err)
			slog.Warn("Kafka transaction validation failed (REJECTED)",
				"error", err,
				"error_code", meta.Code,
				"reason", err.Error(),
				"dto", dto,
				"raw_payload", string(msg.Value),
				"offset", msg.Offset,
			)
			continue
		}

		if dto.Timestamp == "" {
			slog.Warn(MsgMissingZeroTimestamp, "userID", dto.UserID, "offset", msg.Offset)
		}

		processCtx, cancel := context.WithTimeout(ctx, time.Duration(c.cfg.ProcessTimeoutMs)*time.Millisecond)

		err = c.svc.RegisterTransaction(processCtx, t)
		cancel()

		if err != nil {
			meta := boundary.Classify(err)
			slog.Error(MsgFailedToProcessTransaction, "error", err, "error_code", meta.Code, "userID", dto.UserID)
			c.pauseWithJitter(ctx)
			continue
		}

		slog.Debug(MsgTransactionProcessed, "userID", dto.UserID)
	}
}

func (c *Consumer) pauseWithJitter(ctx context.Context) {
	cfg := withDefaultKafkaConfig(c.cfg)
	delay := time.Duration(cfg.RetryBaseDelayMs)*time.Millisecond + time.Duration(rand.Int63n(int64(time.Duration(cfg.RetryJitterMs)*time.Millisecond)))
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func withDefaultKafkaConfig(cfg config.Kafka) config.Kafka {
	if cfg.ProcessTimeoutMs <= 0 {
		cfg.ProcessTimeoutMs = int(ProcessTransactionTimeout / time.Millisecond)
	}
	if cfg.RetryBaseDelayMs <= 0 {
		cfg.RetryBaseDelayMs = int(RetryBackoffBaseDelay / time.Millisecond)
	}
	if cfg.RetryJitterMs <= 0 {
		cfg.RetryJitterMs = int(RetryBackoffMaxJitter / time.Millisecond)
	}
	return cfg
}
