package kafka

import (
	"casino-transaction-system/internal/service"
	"context"
	"encoding/json"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	svc    service.TransactionService
}

func NewConsumer(brokers []string, topic, groupID string, svc service.TransactionService) *Consumer {
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
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	slog.Info("Starting Kafka consumer loop...")
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info(MsgKafkaShuttingDown)
				return c.reader.Close()
			}
			slog.Error(MsgFailedToReadMessage, "error", err)
			continue
		}

		slog.Debug(MsgKafkaMessageReceived, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)

		var dto TransactionDTO
		if err := json.Unmarshal(msg.Value, &dto); err != nil {
			slog.Error(MsgFailedToUnmarshalMessage, "error", err, "value", string(msg.Value))
			continue
		}

		if dto.Timestamp == "" {
			slog.Warn(MsgMissingZeroTimestamp, "userID", dto.UserID, "offset", msg.Offset)
		}

		// Separate context for processing
		processCtx, cancel := context.WithTimeout(context.Background(), ProcessTransactionTimeout)
		
		err = c.svc.RegisterTransaction(processCtx, dto.ToDomain())
		cancel() 

		if err != nil {
			slog.Error(MsgFailedToProcessTransaction, "error", err, "userID", dto.UserID)
			continue
		}

		slog.Debug(MsgTransactionProcessed, "userID", dto.UserID)
	}
}
