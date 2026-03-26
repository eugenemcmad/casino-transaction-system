# Testing guidelines

**How to run tests** (commands, Docker, coverage, Make): [`DEVELOPMENT.md`](DEVELOPMENT.md#testing).

## Test types

- **Unit:** default build, no tags.
- **Integration:** `//go:build integration`, files `*_integration_test.go`.
- **E2E:** `//go:build e2e`, files under `e2e/` (e.g. `*_e2e_test.go`). E2E tests **fail** (exit non-zero) if Docker/testcontainers cannot start; integration tests without the `e2e` tag still **skip** when testcontainers are unavailable.

## Naming

- Functions: `Test<Subject>_<Behavior>`.
- Subtests (`t.Run`): `ok/...` and `err/...`.
- Table-driven helpers: `cases`, `tc`, `want`, `got`, `wantErr`.

## Style

- Mocks of `service.TransactionService` must implement `RegisterTransaction` and `RegisterTransactions` (Kafka processor may call bulk inserts).
- **Table-driven** tests for many input/output pairs.
- **Scenario-style** for heavy setup (`sqlmock`, containers, full flows).
- One concern per test function.

## Assertions

- Prefer behavior over implementation details.
- Clear failures (expected vs actual).

## Logging

- Use `t.Log` / `t.Logf`, not `fmt.Println`.
- No emoji in test output.

## Coverage

Target for the task: **≥ 85%** statement coverage where practical.
