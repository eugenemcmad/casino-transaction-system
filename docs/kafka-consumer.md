# Kafka processor consumer

The processor (`cmd/processor`) uses `internal/transport/kafka.Consumer`, which **processes messages strictly one after another** in a single goroutine: read → decode → `RegisterTransaction` → repeat.

## Why not parallelize inside one consumer?

- **Offset commits and `kafka-go`:** With the current `ReadMessage` loop, advancing to the next message is tied to finishing the current iteration. Spawning workers without a deliberate commit strategy risks committing offsets **before** persistence succeeds, which weakens delivery guarantees on crash.
- **Ordering:** Kafka preserves order **per partition**. Parallel handling of the same partition can apply events out of order. For this service, inserts are independent rows with `ON CONFLICT DO NOTHING`, but any future logic that assumes per-partition or per-key ordering would break unless work is partitioned (e.g. by key) or commits are coordinated manually.

## How to scale throughput

Prefer **horizontal scaling** aligned with Kafka:

1. **More topic partitions** (and message keys if you need user-local ordering).
2. **More processor instances** in the same consumer group so partitions are split across consumers.

That increases parallelism **without** changing commit semantics in this binary. Tune PostgreSQL pool settings if many instances contend on the same database.

## If you need in-process parallelism later

You would typically introduce an explicit pipeline: decode → bounded worker pool → **commit offsets only after successful `RegisterTransaction`**, with per-partition ordering rules and tests for failure modes. That is intentionally out of scope for the current implementation.
