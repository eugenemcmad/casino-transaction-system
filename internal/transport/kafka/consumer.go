// Package kafka implements a Kafka consumer that decodes DTOs and registers domain transactions.
//
// Processing is partition-aware asynchronous: messages from different partitions are processed in
// parallel, while each partition remains strictly sequential so commit order stays aligned with persistence.
// See docs/kafka-consumer.md for scaling and reliability notes.
package kafka

import (
	"casino-transaction-system/internal/boundary"
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/domain"
	basemetrics "casino-transaction-system/internal/observability/metrics"
	"casino-transaction-system/internal/service"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
)

type messageReader interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
	Close() error
}

type messageWriter interface {
	WriteMessages(context.Context, ...kafka.Message) error
	Close() error
}

// Consumer reads JSON transaction messages and dispatches them to TransactionService.
type Consumer struct {
	reader   messageReader
	writer   messageWriter
	svc      service.TransactionService
	cfg      config.Kafka
	metrics  MetricsSink
	now      func() time.Time
	inflight int64
}

// NewConsumer builds a kafka-go reader and consumer with retry/backoff settings from cfg.
func NewConsumer(brokers []string, topic, groupID string, svc service.TransactionService, cfg config.Kafka) *Consumer {
	return NewConsumerWithMetrics(brokers, topic, groupID, svc, cfg, newLogMetricsSink())
}

// NewConsumerWithMetrics builds a kafka-go reader/consumer and reuses provided metrics sink.
func NewConsumerWithMetrics(
	brokers []string,
	topic, groupID string,
	svc service.TransactionService,
	cfg config.Kafka,
	metricsSink basemetrics.Sink,
) *Consumer {
	cfg = normalizeKafkaConfig(cfg)
	if metricsSink == nil {
		metricsSink = newLogMetricsSink()
	}
	slog.Debug("Initializing Kafka Consumer", "brokers", brokers, "topic", topic, "groupID", groupID, "dlq_topic", cfg.DLQTopic)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       DefaultMinBytes,
		MaxBytes:       DefaultMaxBytes,
		CommitInterval: 0, // Explicit manual commit after successful persistence
	})

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        cfg.DLQTopic,
		RequiredAcks: kafka.RequireAll,
	}

	return &Consumer{
		reader:  reader,
		writer:  writer,
		svc:     svc,
		cfg:     cfg,
		metrics: metricsSink,
		now:     time.Now,
	}
}

// Start blocks reading messages until ctx is cancelled or the reader fails permanently.
//
// Concurrency model:
//   - Producers are decoupled from DB latency (async at system level).
//   - The fetch loop reads from Kafka and hands each message to a per-partition worker goroutine.
//   - Different partitions are processed in parallel; within one partition, messages are strictly
//     sequential, preserving Kafka's per-partition ordering and keeping commits aligned with handling.
func (c *Consumer) Start(ctx context.Context) error {
	slog.Info("Starting Kafka consumer loop...")
	c.cfg = normalizeKafkaConfig(c.cfg)

	// Wrap context for internal fatal-error driven graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var fatalErr error
	var fatalOnce sync.Once
	triggerFatal := func(err error) {
		fatalOnce.Do(func() {
			fatalErr = err
			cancel()
		})
	}

	flushTicker := time.NewTicker(time.Duration(c.cfg.MetricsFlushSec) * time.Second)
	defer flushTicker.Stop()

	// Run metrics flusher in the background
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.metrics.Flush()
				return
			case <-flushTicker.C:
				c.metrics.Flush()
			}
		}
	}()

	var workers sync.Map // Map of partitionID (int) -> chan kafka.Message
	var workersWG sync.WaitGroup

	// commitCtx survives ctx cancellation so DLQ routing and offset commits can finish while the
	// process is winding down; shutdown is still bounded by ShutdownDrainTimeoutSec on worker Wait().
	commitCtx := context.WithoutCancel(ctx)

	// Graceful shutdown orchestrator
	defer func() {
		slog.Info(MsgKafkaShuttingDown)

		// Stop accepting new messages by closing all partition channels
		workers.Range(func(key, value any) bool {
			ch := value.(chan kafka.Message)
			close(ch)
			return true
		})

		// Wait for partition workers to drain (bounded to avoid hanging forever on DLQ/commit outages).
		drainDone := make(chan struct{})
		go func() {
			workersWG.Wait()
			close(drainDone)
		}()
		drainTimeout := time.Duration(c.cfg.ShutdownDrainTimeoutSec) * time.Second
		select {
		case <-drainDone:
		case <-time.After(drainTimeout):
			slog.Warn("Kafka consumer: partition workers drain timed out; closing reader",
				"timeout_sec", c.cfg.ShutdownDrainTimeoutSec)
		}

		// Close infrastructure
		c.closeWriter()
		if err := c.reader.Close(); err != nil {
			slog.Error("Failed to close Kafka reader cleanly", "error", err)
		}
		slog.Info("Kafka consumer shutdown complete.")
	}()

	for {
		// FetchMessage reads the next message without automatically committing its offset.
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return fatalErr // Graceful exit requested by context (or internal fatal error)
			}
			meta := boundary.Classify(err)
			slog.Error(MsgFailedToReadMessage, "error", err, "error_code", meta.Code)
			c.metrics.IncCounter(MetricMessagesTotal, metricLabels{"result": "failed", "reason": "fetch"}, 1)
			c.pauseWithJitter(ctx)
			continue
		}

		// Dispatch message to a dedicated sequential worker for this partition.
		// This guarantees that ordering per partition is strictly maintained.
		chIntf, loaded := workers.LoadOrStore(msg.Partition, make(chan kafka.Message, PartitionWorkerQueueSize))
		ch := chIntf.(chan kafka.Message)

		if !loaded {
			workersWG.Add(1)
			go c.partitionWorker(ctx, commitCtx, msg.Partition, ch, &workersWG, triggerFatal)
		}

		// Backpressure: blocks FetchMessage when the partition buffer is full.
		// Tradeoff (Head-of-Line Blocking): A slow partition will starve other healthy partitions
		// on this instance. Chosen to ensure bounded memory vs creating per-partition readers.
		select {
		case ch <- msg:
		case <-ctx.Done():
			return fatalErr
		}
	}
}

// partitionWorker processes messages for a single partition sequentially.
func (c *Consumer) partitionWorker(appCtx, commitCtx context.Context, partition int, ch <-chan kafka.Message, wg *sync.WaitGroup, triggerFatal func(error)) {
	defer wg.Done()
	slog.Debug("Started partition worker", "partition", partition)

	interval := time.Duration(c.cfg.BatchFlushIntervalSec) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	batchMsgs := make([]kafka.Message, 0, c.cfg.BatchSize)
	batchDomains := make([]domain.Transaction, 0, c.cfg.BatchSize)

	flush := func() {
		if len(batchMsgs) == 0 {
			return
		}
		c.flushBatch(appCtx, commitCtx, batchMsgs, batchDomains, triggerFatal)
		batchMsgs = batchMsgs[:0]
		batchDomains = batchDomains[:0]
	}

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				// Graceful shutdown signals finish
				return
			}

			if appCtx.Err() != nil {
				slog.Info("Worker stopping due to shutdown, leaving message uncommitted", "partition", partition, "offset", msg.Offset)
				return
			}

			t, err := c.parseAndValidateMsg(msg)
			if err != nil {
				// Preserve strict offset sequence: flush existing batch first
				flush()

				// Discard/DLQ bad message via standard sequential path
				current := atomic.AddInt64(&c.inflight, 1)
				c.metrics.SetGauge(MetricInflightMessages, nil, float64(current))

				c.handleMessage(appCtx, commitCtx, msg, triggerFatal)

				current = atomic.AddInt64(&c.inflight, -1)
				c.metrics.SetGauge(MetricInflightMessages, nil, float64(current))
				continue
			}

			batchMsgs = append(batchMsgs, msg)
			batchDomains = append(batchDomains, t)

			if len(batchMsgs) >= c.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (c *Consumer) flushBatch(appCtx, commitCtx context.Context, batchMsgs []kafka.Message, batchDomains []domain.Transaction, triggerFatal func(error)) {
	if len(batchMsgs) == 0 {
		return
	}

	current := atomic.AddInt64(&c.inflight, int64(len(batchMsgs)))
	c.metrics.SetGauge(MetricInflightMessages, nil, float64(current))
	defer func() {
		current = atomic.AddInt64(&c.inflight, -int64(len(batchMsgs)))
		c.metrics.SetGauge(MetricInflightMessages, nil, float64(current))
	}()

	startedAt := c.now()
	var bulkErr error

	for attempt := 0; attempt <= c.cfg.MaxProcessRetries; attempt++ {
		processCtx, cancel := context.WithTimeout(appCtx, time.Duration(c.cfg.ProcessTimeoutMs)*time.Millisecond)
		bulkErr = c.svc.RegisterTransactions(processCtx, batchDomains)
		cancel()

		if bulkErr == nil {
			break
		}
		if appCtx.Err() != nil {
			return
		}

		meta := boundary.Classify(bulkErr)
		if !meta.Retryable {
			break
		}

		c.metrics.IncCounter(MetricRetriesTotal, metricLabels{"reason": "process_bulk_error"}, 1)
		c.sleepBackoff(appCtx, attempt)
	}

	if bulkErr == nil {
		slog.Debug("Bulk processed valid batch", "count", len(batchMsgs))
		c.metrics.ObserveDuration(MetricProcessingDurationMs, metricLabels{"result": "processed", "mode": "bulk"}, c.now().Sub(startedAt))
		c.metrics.SetGauge(MetricLastSuccessUnix, nil, float64(c.now().Unix()))

		for {
			if err := c.reader.CommitMessages(commitCtx, batchMsgs...); err == nil {
				c.metrics.IncCounter(MetricCommitTotal, metricLabels{"result": "success", "mode": "bulk"}, int64(len(batchMsgs)))
				break
			}
			if appCtx.Err() != nil {
				return
			}
			slog.Error("Failed to bulk commit", "batch_size", len(batchMsgs))
			c.pauseWithJitter(appCtx)
		}
		return
	}

	// Fallback mechanism (Sequential)
	slog.Warn("Bulk insert failed, falling back to sequential processing", "error", bulkErr, "batchSize", len(batchMsgs))

	// We decrement the batch inflight metric because handleMessage manages its own metric for each message.
	atomic.AddInt64(&c.inflight, -int64(len(batchMsgs)))

	for _, msg := range batchMsgs {
		atomic.AddInt64(&c.inflight, 1)
		c.handleMessage(appCtx, commitCtx, msg, triggerFatal)
		atomic.AddInt64(&c.inflight, -1)
	}

	// Restore tracker before defer runs
	atomic.AddInt64(&c.inflight, int64(len(batchMsgs)))
}

func (c *Consumer) handleMessage(appCtx, commitCtx context.Context, msg kafka.Message, triggerFatal func(error)) {
	partStr := strconv.Itoa(msg.Partition)
	labelsFailed := metricLabels{"topic": msg.Topic, "partition": partStr, "result": "failed"}
	labelsSuccess := metricLabels{"topic": msg.Topic, "partition": partStr, "result": "processed"}
	labelsDLQ := metricLabels{"topic": msg.Topic, "partition": partStr, "result": "dlq"}
	labelsFailDur := metricLabels{"result": "failed"}
	labelsProcDur := metricLabels{"result": "processed"}

	c.observeLag(msg)
	startedAt := c.now()

	// Execute Business Logic with Retries (respects app shutdown)
	err := c.processMessageWithRetries(appCtx, msg)
	if err != nil {
		// If failure was caused by application shutdown, ABORT and DO NOT COMMIT.
		// Kafka will re-deliver this message upon restart (At-Least-Once delivery).
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			slog.Info("Processing aborted due to shutdown", "offset", msg.Offset)
			return
		}

		c.metrics.ObserveDuration(MetricProcessingDurationMs, labelsFailDur, c.now().Sub(startedAt))
		c.metrics.IncCounter(MetricMessagesTotal, labelsFailed, 1)

		// Terminal Failure -> Route to Dead Letter Queue (DLQ).
		// CRITICAL: We use commitCtx (detached) and retry indefinitely.
		// If we drop the message here, committing the next offset will implicitly commit this failed one (data loss).
		for {
			dlqErr := c.routeToDLQ(commitCtx, msg, err)
			if dlqErr == nil {
				break
			}
			slog.Error("Failed to route message to DLQ (retrying to prevent offset loss)", "error", dlqErr, "offset", msg.Offset)
			c.metrics.IncCounter(MetricDLQTotal, metricLabels{"result": "error"}, 1)

			var nrErr nonRetryableError
			if errors.As(dlqErr, &nrErr) {
				slog.Error("Fatal DLQ error, initiating graceful shutdown", "error", dlqErr, "offset", msg.Offset)
				// Initiates consumer shutdown securely without dropping the uncommitted message
				triggerFatal(fmt.Errorf("fatal DLQ failure at offset %d: %w", msg.Offset, dlqErr))
				return
			}

			if appCtx.Err() != nil {
				slog.Info("Aborting DLQ retry due to shutdown", "offset", msg.Offset)
				return
			}
			c.pauseWithJitter(appCtx)
		}
		c.metrics.IncCounter(MetricMessagesTotal, labelsDLQ, 1)

	} else {
		// Success metrics
		c.metrics.SetGauge(MetricLastSuccessUnix, nil, float64(c.now().Unix()))
		c.metrics.ObserveDuration(MetricProcessingDurationMs, labelsProcDur, c.now().Sub(startedAt))
		c.metrics.IncCounter(MetricMessagesTotal, labelsSuccess, 1)
	}

	// Commit Offset
	// CRITICAL: We use commitCtx (detached) and retry indefinitely.
	// We only commit after successful processing OR successful DLQ routing.
	for {
		if err := c.commit(commitCtx, msg); err == nil {
			break
		}

		if appCtx.Err() != nil {
			slog.Info("Aborting commit retry due to shutdown", "offset", msg.Offset)
			return
		}
		c.pauseWithJitter(appCtx)
	}
}

func (c *Consumer) parseAndValidateMsg(msg kafka.Message) (domain.Transaction, error) {
	// payloadFingerprint is only used on error logging paths so we do not pay SHA-256 on the success hot path.
	var dto TransactionDTO
	if err := json.Unmarshal(msg.Value, &dto); err != nil {
		payloadHash, payloadSize := payloadFingerprint(msg.Value)
		slog.Error(MsgFailedToUnmarshalMessage,
			"error", err,
			"offset", msg.Offset,
			"payload_sha256", payloadHash,
			"payload_size_bytes", payloadSize,
		)
		return domain.Transaction{}, nonRetryableError{cause: err}
	}

	t, err := dto.ToDomain()
	if err != nil {
		payloadHash, payloadSize := payloadFingerprint(msg.Value)
		meta := boundary.Classify(err)
		slog.Warn("Kafka transaction parse failed",
			"error", err,
			"error_code", meta.Code,
			"offset", msg.Offset,
			"payload_sha256", payloadHash,
			"payload_size_bytes", payloadSize,
		)
		return domain.Transaction{}, nonRetryableError{cause: err}
	}

	if err := t.Validate(); err != nil {
		payloadHash, payloadSize := payloadFingerprint(msg.Value)
		meta := boundary.Classify(err)
		slog.Warn("Kafka transaction validation failed (REJECTED)",
			"error", err,
			"error_code", meta.Code,
			"reason", err.Error(),
			"offset", msg.Offset,
			"payload_sha256", payloadHash,
			"payload_size_bytes", payloadSize,
		)
		return domain.Transaction{}, nonRetryableError{cause: err}
	}

	if dto.Timestamp == "" {
		slog.Warn(MsgMissingZeroTimestamp, "userID", dto.UserID, "offset", msg.Offset)
	}

	return t, nil
}

func (c *Consumer) processMessageWithRetries(ctx context.Context, msg kafka.Message) error {
	t, err := c.parseAndValidateMsg(msg)
	if err != nil {
		return err
	}

	// Retry only transient service-level processing failures (e.g. Database timeouts)
	for attempt := 0; attempt <= c.cfg.MaxProcessRetries; attempt++ {
		// Apply timeout to the specific DB operation, but inherit cancellation from main ctx
		processCtx, cancel := context.WithTimeout(ctx, time.Duration(c.cfg.ProcessTimeoutMs)*time.Millisecond)
		err = c.svc.RegisterTransaction(processCtx, t)
		cancel()

		if err == nil {
			slog.Debug(MsgTransactionProcessed, "userID", t.UserID, "attempt", attempt+1)
			return nil
		}

		// If the entire application is shutting down, abort retries immediately
		if ctx.Err() != nil {
			return ctx.Err()
		}

		meta := boundary.Classify(err)
		if !meta.Retryable {
			slog.Error("Aborting retries for non-retryable error", "error", err, "error_code", meta.Code, "userID", t.UserID)
			return nonRetryableError{cause: err}
		}

		slog.Error(MsgFailedToProcessTransaction, "error", err, "error_code", meta.Code, "userID", t.UserID, "attempt", attempt+1)

		if attempt == c.cfg.MaxProcessRetries {
			return fmt.Errorf("process failed after %d attempts: %w", attempt+1, err)
		}

		c.metrics.IncCounter(MetricRetriesTotal, metricLabels{"reason": "process_error"}, 1)
		c.metrics.IncCounter(MetricMessagesTotal, metricLabels{"result": "retried"}, 1)
		c.sleepBackoff(ctx, attempt)
	}

	return nil
}

func (c *Consumer) routeToDLQ(ctx context.Context, msg kafka.Message, processErr error) error {
	if c.writer == nil {
		return nonRetryableError{cause: errors.New("dlq writer is not configured")}
	}

	// Preserve original Kafka metadata for forensic analysis in DLQ consumers
	payloadHash, payloadSize := payloadFingerprint(msg.Value)
	sourceTimeStr := ""
	if !msg.Time.IsZero() {
		sourceTimeStr = msg.Time.UTC().Format(time.RFC3339Nano)
	}

	payload := map[string]any{
		"topic":            msg.Topic,
		"partition":        msg.Partition,
		"offset":           msg.Offset,
		"key":              string(msg.Key),
		"value":            "[REDACTED]",
		"value_sha256":     payloadHash,
		"value_size_bytes": payloadSize,
		"error":            processErr.Error(),
		"failed_at":        c.now().UTC().Format(time.RFC3339Nano),
		"headers":          msg.Headers,
		"sourceTime":       sourceTimeStr,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if err := c.writer.WriteMessages(ctx, kafka.Message{Key: msg.Key, Value: raw}); err != nil {
		return err
	}
	c.metrics.IncCounter(MetricDLQTotal, metricLabels{"result": "success"}, 1)
	return nil
}

func (c *Consumer) commit(ctx context.Context, msg kafka.Message) error {
	if err := c.reader.CommitMessages(ctx, msg); err != nil {
		slog.Error("Failed to commit Kafka offset", "error", err, "offset", msg.Offset, "partition", msg.Partition)
		c.metrics.IncCounter(MetricCommitTotal, metricLabels{"result": "error"}, 1)
		return err
	}
	c.metrics.IncCounter(MetricCommitTotal, metricLabels{"result": "success"}, 1)
	return nil
}

func (c *Consumer) observeLag(msg kafka.Message) {
	if msg.Time.IsZero() {
		return
	}
	lagMs := c.now().Sub(msg.Time).Milliseconds()
	if lagMs < 0 {
		lagMs = 0
	}
	c.metrics.SetGauge(MetricLag, metricLabels{"topic": msg.Topic, "partition": strconv.Itoa(msg.Partition)}, float64(lagMs))
}

func (c *Consumer) sleepBackoff(ctx context.Context, attempt int) {
	// Exponential backoff with jitter prevents synchronized retry storms
	base := float64(c.cfg.RetryBaseDelayMs) * math.Pow(2, float64(attempt))
	delay := time.Duration(base)*time.Millisecond + time.Duration(rand.Int63n(int64(time.Duration(c.cfg.RetryJitterMs)*time.Millisecond)))
	if delay > MaxRetryDelay {
		delay = MaxRetryDelay
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (c *Consumer) pauseWithJitter(ctx context.Context) {
	c.sleepBackoff(ctx, 0)
}

func (c *Consumer) closeWriter() {
	if c.writer == nil {
		return
	}
	if err := c.writer.Close(); err != nil {
		slog.Warn("Failed to close DLQ writer", "error", err)
	}
}

type nonRetryableError struct {
	cause error
}

func (e nonRetryableError) Error() string {
	return e.cause.Error()
}

func payloadFingerprint(raw []byte) (hash string, sizeBytes int) {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), len(raw)
}

func normalizeKafkaConfig(cfg config.Kafka) config.Kafka {
	if cfg.ProcessTimeoutMs <= 0 {
		cfg.ProcessTimeoutMs = int(ProcessTransactionTimeout / time.Millisecond)
	}
	if cfg.RetryBaseDelayMs <= 0 {
		cfg.RetryBaseDelayMs = int(RetryBackoffBaseDelay / time.Millisecond)
	}
	if cfg.RetryJitterMs <= 0 {
		cfg.RetryJitterMs = int(RetryBackoffMaxJitter / time.Millisecond)
	}
	if cfg.MaxProcessRetries < 0 {
		cfg.MaxProcessRetries = DefaultMaxProcessRetries
	}
	if cfg.DLQTopic == "" && cfg.Topic != "" {
		cfg.DLQTopic = cfg.Topic + DefaultDLQTopicSuffix
	}
	if cfg.MetricsFlushSec <= 0 {
		cfg.MetricsFlushSec = DefaultMetricsFlushSec
	}
	if cfg.ShutdownDrainTimeoutSec <= 0 {
		cfg.ShutdownDrainTimeoutSec = DefaultShutdownDrainTimeoutSec
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultBatchSize
	}
	if cfg.BatchFlushIntervalSec <= 0 {
		cfg.BatchFlushIntervalSec = DefaultBatchFlushIntervalSec
	}
	return cfg
}
