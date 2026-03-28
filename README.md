# Casino Transaction System

Go service that ingests **bet/win** transactions from **Kafka**, stores them in **PostgreSQL**, and exposes a **JSON HTTP API** to query history with optional filters.

## Documentation

| Doc | Contents |
|-----|----------|
| **[`DEVELOPMENT.md`](DEVELOPMENT.md)** | Clone, build, migrations, run (Docker Compose or local binaries), tests, coverage, Make targets |
| **[`TESTING.md`](TESTING.md)** | Test types, naming, style (not how to run — see `DEVELOPMENT.md`) |
| **[`docs/kafka-consumer.md`](docs/kafka-consumer.md)** | Processor consumer: sequential processing, scaling (partitions / replicas), why no in-process worker pool |

---

## Quick prerequisites

- **Go** 1.25+ (`go.mod`)
- **Docker** — Compose-based run and integration/E2E tests (testcontainers)
- **PostgreSQL 15+** — only if you run the apps outside Compose without your own DB image

---

## Configuration

Configuration is loaded from an optional YAML file, then **environment variables override** (see `internal/config`).

| Source | Role |
|--------|------|
| `CONFIG_PATH` | Path to YAML (default: `config.yaml` in working directory). If the file is missing, defaults + env are used. |
| Env vars | Override YAML fields (e.g. `PG_URL`, `HTTP_PORT`). |

**Minimum env for a working process:** `APP_NAME`, `APP_VERSION`, `PG_URL`, `KAFKA_BROKERS`, `KAFKA_TOPIC` (Kafka fields are required by config loading even for the API binary).

Useful variables (non-exhaustive):

| Variable | Purpose |
|----------|---------|
| `HTTP_PORT` | API listen port (default `8080`) |
| `HTTP_READ_HEADER_TIMEOUT_SECONDS`, `HTTP_READ_TIMEOUT_SECONDS`, `HTTP_WRITE_TIMEOUT_SECONDS`, `HTTP_IDLE_TIMEOUT_SECONDS` | HTTP server timeouts |
| `PG_URL` | PostgreSQL DSN |
| `PG_POOL_MAX_OPEN`, `PG_POOL_MAX_IDLE`, `PG_CONN_MAX_LIFETIME_MINUTES` | Connection pool |
| `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_GROUP_ID` | Kafka consumer |
| `KAFKA_PROCESS_TIMEOUT_MS`, `KAFKA_RETRY_BASE_DELAY_MS`, `KAFKA_RETRY_JITTER_MS` | Consumer processing / backoff |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` |

Example:

```bash
export APP_NAME=casino-api APP_VERSION=1.0.0
export PG_URL="postgres://postgres:pass@localhost:5432/casino?sslmode=disable"
export KAFKA_BROKERS=localhost:9094
export KAFKA_TOPIC=transactions
go run ./cmd/api
```

Copy [`config.yaml`](config.yaml) and adjust URLs for your environment.

**Run the full stack, migrations, and tests:** [`DEVELOPMENT.md`](DEVELOPMENT.md).

---

## HTTP API (summary)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness — `200` and body `OK` |
| `GET` | `/transactions` | JSON list; optional `user_id` (positive integer), `transaction_type` (`bet` or `win`) |

**Money:** amounts are stored and returned as **integers** (minor units). Kafka/ingress JSON may send `amount` as string or number; parsing is handled in the consumer/DTO layer.

More examples: [`api/tests.http`](api/tests.http).

---

## Repository layout (short)

| Path | Role |
|------|------|
| `cmd/api`, `cmd/processor` | Entrypoints |
| `internal/bootstrap` | Wiring (DB, HTTP, Kafka) |
| `internal/domain` | Domain model and ports |
| `internal/service` | Use cases |
| `internal/repository` | PostgreSQL adapter |
| `internal/transport/http`, `internal/transport/kafka` | Adapters |
| `migrations/` | SQL migrations |
| `pkg/money`, `pkg/timeutil` | Shared parsing helpers |
| `e2e/` | End-to-end tests (`go:build e2e`) |

**Compose:** dev stack [`docker-compose.dev.yaml`](docker-compose.dev.yaml); production-style (external DB/Kafka) [`docker-compose.prod.yaml`](docker-compose.prod.yaml) — set `PROD_PG_URL` and `PROD_KAFKA_BROKERS`.
