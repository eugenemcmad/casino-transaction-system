# Development — build, run, test

Checklist for running and verifying the project locally. Test **style** and naming: [`TESTING.md`](TESTING.md). Runtime **settings**: [`README.md`](../README.md#configuration).

---

## Prerequisites

- **Go** 1.25+ (see `go.mod`; toolchain may download automatically)
- **Docker** — `docker compose` for the dev stack; **daemon required** for integration and E2E (`testcontainers`)
- **PostgreSQL 15+** — if you run API/processor on the host without Compose-provided Postgres
- **Docker Compose V2** — dev `docker-compose.dev.yaml` uses `service_completed_successfully` on the migrate job
- On Windows, use **Docker Desktop** (Linux containers) for the same flows

---

## Clone and modules

```bash
git clone <repository-url>
cd casino-transaction-system
go mod download
```

(`go build` / `go test` will fetch modules if you skip this step.)

---

## Build

With Make (Linux/macOS; Git Bash / WSL on Windows):

```bash
make build-api build-processor
```

Output: `bin/api`, `bin/processor`.

Without Make:

```bash
mkdir -p bin
go build -o bin/api ./cmd/api
go build -o bin/processor ./cmd/processor
```

---

## Database migrations

SQL files are in [`migrations/`](../migrations/). In dev Compose, migrations run automatically via the `migrate` service.

**Local [`migrate`](https://github.com/golang-migrate/migrate) CLI:**

```bash
migrate -path migrations -database "postgres://postgres:pass@localhost:5432/casino?sslmode=disable" up
```

**Manual migrate container** (if you use the dev compose file): `make migrate-up`.

---

## Run

### Option A — Docker Compose (recommended)

Builds images, starts Postgres and Kafka, runs migrations, then API and processor:

```bash
docker compose -f docker-compose.dev.yaml up -d --build
```

- API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Stop and drop volumes:

  ```bash
  docker compose -f docker-compose.dev.yaml down -v
  ```

**Make:** `make docker-up` / `make docker-down` / `make docker-logs` (Makefile uses the `docker-compose` binary name; use `docker compose` if you rely on the plugin instead).

### Option B — local processes

1. Start **PostgreSQL** and **Kafka**.
2. Apply **migrations** (see above).
3. Create the Kafka **topic** (name must match `KAFKA_TOPIC`).
4. Set **environment variables** — [README → Configuration](../README.md#configuration).
5. `go run ./cmd/processor`
6. `go run ./cmd/api` (second terminal)

**Make:** `make run-processor`, `make run-api`.

---

## Testing

| Scope | Command | Docker |
|-------|---------|--------|
| Unit (default) | `go test ./...` | Not required for the default package set |
| + Integration | `go test -tags=integration ./...` | Required (testcontainers) |
| + E2E | `go test -tags=e2e ./...` | Required; slower full-stack run |

**Make:** `make test`, `make test-integration`, `make test-e2e`.

**Race detector** (optional, needs CGO): `make test-race`.

**Coverage (statements):**

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func coverage.out
```

Or `make cover`.

**Coverage target for the task:** ≥ 85% — see [`TESTING.md`](TESTING.md).

---

## Make targets

Run `make help` for the full list. Common:

| Target | Action |
|--------|--------|
| `build-api`, `build-processor` | Binaries under `bin/` |
| `test`, `test-integration`, `test-e2e`, `test-race` | Tests |
| `cover` | Coverage report |
| `docker-up`, `docker-down`, `docker-logs` | Dev Compose |
| `migrate-up` | Run migrate service via dev Compose |
| `run-api`, `run-processor` | `"go run"` entrypoints |
| `lint` | `golangci-lint` (if installed) |

---

## Lint (optional)

```bash
make lint
```

Requires [`golangci-lint`](https://golangci-lint.run/).
