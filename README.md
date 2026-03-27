# Casino Transaction System

Go service that ingests **bet/win** transactions from **Kafka**, stores them in **PostgreSQL**, and exposes a **JSON HTTP API** to query history with optional filters.

---

## Prerequisites

- **Go** 1.25+ (see `go.mod`; toolchain may download automatically)
- **Docker Desktop** (or Docker Engine) — for Compose, integration tests, and E2E tests that use testcontainers
- **PostgreSQL** 15+ — if you run binaries locally without Compose

---

## Build, run, and test

For a **step-by-step checklist** (clone, compile binaries, Docker Compose vs local processes, unit / integration / E2E tests, coverage, optional lint), see **[`DEVELOPMENT.md`](DEVELOPMENT.md)**.

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

---

## Database migrations

SQL files live in [`migrations/`](migrations/). Apply them with [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate) or use the **migrate** service in Docker Compose (see below).

Example (local `migrate` CLI):

```bash
migrate -path migrations -database "postgres://postgres:pass@localhost:5432/casino?sslmode=disable" up
```

---

## Run locally (two processes)

1. Start **PostgreSQL** and **Kafka**, apply migrations, create the topic if needed.
2. **Processor** (consumes Kafka, writes DB):

   ```bash
   go run ./cmd/processor
   ```

3. **API** (queries DB):

   ```bash
   go run ./cmd/api
   ```

---

## Run with Docker Compose (dev stack)

Builds API and processor images, starts Postgres, Kafka, runs migrations, then apps:

```bash
docker compose -f docker-compose.dev.yaml up -d --build
```

- API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Tear down (including volumes):

  ```bash
  docker compose -f docker-compose.dev.yaml down -v
  ```

Requires **Docker Compose V2** (for `service_completed_successfully` on the migrate job).

Production-style compose (external DB/Kafka): [`docker-compose.prod.yaml`](docker-compose.prod.yaml) — set `PROD_PG_URL` and `PROD_KAFKA_BROKERS` in the environment.

---

## HTTP API (summary)

| Method | Path | Description |
|--------|------|----------------|
| `GET` | `/health` | Liveness — `200` and body `OK` |
| `GET` | `/transactions` | JSON list; optional `user_id` (positive integer), `transaction_type` (`bet` or `win`) |

**Money:** amounts are stored and returned as **integers** (minor units). Kafka/ingress JSON may send `amount` as string or number; parsing is handled in the consumer/DTO layer.

More examples: [`api/tests.http`](api/tests.http).

---

## Testing

| Command | What runs |
|---------|-----------|
| `go test ./...` | Unit tests (default) |
| `go test -tags=integration ./...` | Unit + integration (Docker/testcontainers) |
| `go test -tags=e2e ./...` | Includes E2E (Kafka + Postgres + API; long-running) |

Same commands via Make: `make test`, `make test-integration`, `make test-e2e`. Full workflow and coverage commands: [`DEVELOPMENT.md`](DEVELOPMENT.md#test).

Coverage (statement):

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func coverage.out
```

Conventions: [`TESTING.md`](TESTING.md). Target coverage per task: **≥ 85%**.

Integration/E2E need a working **Docker** daemon (Linux containers; on Windows use Docker Desktop).

---

## Makefile shortcuts

On Linux/macOS (Git Bash / WSL), `make help` lists targets. Examples:

- `make build-api` / `make build-processor` — binaries under `bin/`
- `make test` / `make test-integration` / `make test-e2e`
- `make cover` — coverage report
- `make docker-up` / `make docker-down` — dev Compose (uses `docker-compose` CLI name; use `docker compose` if you prefer the plugin)

---

## Repository layout (short)

| Path | Role |
|------|------|
| `DEVELOPMENT.md` | Build / run / test checklist for reviewers |
| `TESTING.md` | Test naming and style |
| `cmd/api`, `cmd/processor` | Entrypoints |
| `internal/bootstrap` | Wiring (DB, HTTP, Kafka) |
| `internal/domain` | Domain model and ports |
| `internal/service` | Use cases |
| `internal/repository` | PostgreSQL adapter |
| `internal/transport/http`, `internal/transport/kafka` | Adapters |
| `migrations/` | SQL migrations |
| `pkg/money`, `pkg/timeutil` | Shared parsing helpers |
| `e2e/` | End-to-end tests (`go:build e2e`) |

