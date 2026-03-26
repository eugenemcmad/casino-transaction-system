package kafka

import "time"

const (
	// Kafka Reader configuration defaults
	DefaultMinBytes          = 10e3 // 10KB
	DefaultMaxBytes          = 10e6 // 10MB
	PartitionWorkerQueueSize = 128

	// Timeouts
	ProcessTransactionTimeout      = 5 * time.Second
	RetryBackoffBaseDelay          = 100 * time.Millisecond
	RetryBackoffMaxJitter          = 300 * time.Millisecond
	MaxRetryDelay                  = 5 * time.Second
	DefaultMaxProcessRetries       = 3
	DefaultMetricsFlushSec         = 15
	DefaultDLQTopicSuffix          = ".dlq"
	DefaultShutdownDrainTimeoutSec = 60
	DefaultBatchSize               = 100
	DefaultBatchFlushIntervalSec   = 5

	// Log messages
	MsgKafkaShuttingDown          = "Kafka consumer shutting down..."
	MsgFailedToReadMessage        = "Failed to read message from Kafka"
	MsgFailedToUnmarshalMessage   = "Failed to unmarshal Kafka message"
	MsgMissingZeroTimestamp       = "Received Kafka message with missing/zero timestamp"
	MsgFailedToProcessTransaction = "Failed to process transaction"
	MsgTransactionProcessed       = "Transaction processed successfully from Kafka"

	// Prometheus-compatible metric names.
	MetricMessagesTotal        = "kafka_consumer_messages_total"
	MetricProcessingDurationMs = "kafka_consumer_processing_duration_ms"
	MetricLag                  = "kafka_consumer_lag"
	MetricRetriesTotal         = "kafka_consumer_retries_total"
	MetricDLQTotal             = "kafka_consumer_dlq_total"
	MetricCommitTotal          = "kafka_consumer_commit_total"
	MetricInflightMessages     = "consumer_inflight_messages"
	MetricLastSuccessUnix      = "consumer_last_success_unix"
)
