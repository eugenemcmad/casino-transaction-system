# Kafka processor consumer

The processor (`cmd/processor`) uses `internal/transport/kafka.Consumer` with **partition-aware asynchronous processing**:

- messages from different partitions are handled in parallel (one worker goroutine per partition)
- within a partition, messages are taken in order; **valid** payloads are buffered and written in **batches**
- invalid payloads are handled immediately on the sequential path (DLQ + commit), without blocking the batch buffer
- offsets are committed after successful persistence (bulk commit for a batch, or per message on fallback/DLQ path)

## Per-partition batching

Each partition worker keeps an in-memory buffer of decoded, validated transactions. The buffer is flushed when:

- its size reaches **`Kafka.BatchSize`** (default `100`), or
- **`Kafka.BatchFlushIntervalSec`** (default `5`) elapses between ticks

On flush, the consumer calls **`RegisterTransactions`** (bulk insert). If bulk fails after retryable errors are exhausted, it **falls back to sequential** `RegisterTransaction` per message (same partition), then commits offsets accordingly.

Configure via `KAFKA_BATCH_SIZE` and `KAFKA_BATCH_FLUSH_INTERVAL_SEC` (see `internal/config`).

**Fetch path:** a single goroutine calls `FetchMessage` and sends to the partition channel. If one partition’s channel is full, fetch blocks for the whole consumer (head-of-line blocking), which bounds memory at the cost of cross-partition fairness on that instance.

## Why this async model?

- **Commit safety:** `FetchMessage` + explicit `CommitMessages` keeps commit timing under control.
- **Ordering safety:** Kafka order is preserved per partition because each partition has its own sequential worker.
- **Real concurrency:** throughput scales with partition count and active consumer instances.

## How to scale throughput

Prefer **horizontal scaling** aligned with Kafka:

1. **More topic partitions** (and message keys if you need user-local ordering).
2. **More processor instances** in the same consumer group so partitions are split across consumers.

That increases parallelism **without** changing commit semantics in this binary. Tune PostgreSQL pool settings if many instances contend on the same database.

## Shutdown drain timeout behavior

Graceful shutdown is bounded by `Kafka.ShutdownDrainTimeoutSec`:

- on shutdown, the consumer stops taking new partition work and waits for active partition workers
- if workers do not finish within the timeout, shutdown proceeds (reader/writer close)
- messages that were fetched but not committed before timeout are intentionally left uncommitted and
  will be re-delivered after restart (at-least-once semantics)

This prevents the process from hanging indefinitely when Kafka/DLQ infrastructure is degraded.

## If you need more in-process parallelism later

You could add a pipeline stage (e.g. decode → bounded pool) **per partition** while keeping commits ordered. The current design favors one sequential worker per partition plus batching, which keeps retry/DLQ/commit semantics easier to reason about.
