package kafka

import "time"

const (
	// Kafka Reader configuration defaults
	DefaultMinBytes = 10e3 // 10KB
	DefaultMaxBytes = 10e6 // 10MB

	// Timeouts
	ProcessTransactionTimeout = 5 * time.Second
	RetryBackoffBaseDelay     = 100 * time.Millisecond
	RetryBackoffMaxJitter     = 300 * time.Millisecond

	// Log messages
	MsgKafkaShuttingDown          = "Kafka consumer shutting down..."
	MsgKafkaMessageReceived       = "Kafka message received"
	MsgFailedToReadMessage        = "Failed to read message from Kafka"
	MsgFailedToUnmarshalMessage   = "Failed to unmarshal Kafka message"
	MsgMissingZeroTimestamp       = "Received Kafka message with missing/zero timestamp"
	MsgFailedToProcessTransaction = "Failed to process transaction"
	MsgTransactionProcessed       = "Transaction processed successfully from Kafka"
)
