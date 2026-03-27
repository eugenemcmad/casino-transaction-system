# Development — build, run, test

Use this as a checklist when reviewing or verifying the project locally. Style and naming rules for tests live in [`TESTING.md`](TESTING.md).

---

## Prerequisites

Same as [`README.md`](README.md#prerequisites): **Go** 1.25+, **Docker** (for Compose, integration tests, E2E), and optionally local **PostgreSQL** if you run binaries without Compose.

---

## Get the code and dependencies

```bash
git clone <repository-url>
cd casino-transaction-system
go mod download
```

(`go build` / `go test` will fetch modules if you skip `download`.)

---

## Build

Compile both entrypoints:

```bash
make build-api build-processor
```

Binaries: `bin/api`, `bin/processor`.

Without Make:

```bash
mkdir -p bin
go build -o bin/api ./cmd/api
go build -o bin/processor ./cmd/processor
```

---

## Run

### Option A — Docker Compose (simplest)

Starts PostgreSQL, Kafka, runs migrations, then API and processor:

```bash
docker compose -f docker-compose.dev.yaml up -d --build
```

- API: `http://localhost:8080`
- Swagger: `http://localhost:8080/swagger/index.html`
- Tear down (including volumes): `docker compose -f docker-compose.dev.yaml down -v`

`Makefile`: `make docker-up` / `make docker-down` (uses `docker-compose` CLI; use `docker compose` if you use the Compose plugin).

### Option B — local processes

1. Run **PostgreSQL** and **Kafka** (versions as in README).
2. Apply **migrations** (see [Database migrations](README.md#database-migrations); local `migrate` CLI or any compatible tool).
3. Ensure the Kafka **topic** exists (name must match `KAFKA_TOPIC` in config).
4. Set **environment variables** as in [Configuration](README.md#configuration) (minimum: `APP_NAME`, `APP_VERSION`, `PG_URL`, `KAFKA_BROKERS`, `KAFKA_TOPIC`).
5. In one terminal: `go run ./cmd/processor`
6. In another: `go run ./cmd/api`

`Makefile` shortcuts: `make run-processor`, `make run-api`.

---

## Test

| What | Command | Notes |
|------|---------|--------|
| Unit tests only | `go test ./...` | No Docker required for the default test set. |
| Unit + integration | `go test -tags=integration ./...` | Uses testcontainers — **Docker daemon** must be running. |
| Unit + integration + E2E | `go test -tags=e2e ./...` | Full stack in containers — **Docker**, can be slow. |

`Makefile`: `make test`, `make test-integration`, `make test-e2e`.

**Race detector** (optional, needs CGO): `make test-race`.

**Coverage** (statement):

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func coverage.out
```

Or `make cover`.

Target coverage for the task: **≥ 85%** (see [`TESTING.md`](TESTING.md)).

---

## Lint (optional)

If [`golangci-lint`](https://golangci-lint.run/) is installed:

```bash
make lint
```
