# Testing guidelines

**How to run tests** (commands, Docker, coverage, Make): [`DEVELOPMENT.md`](DEVELOPMENT.md#testing).

## Test types

- **Unit:** default build, no tags.
- **Integration:** `//go:build integration`, files `*_integration_test.go`.
- **E2E:** `//go:build e2e`, files under `e2e/` (e.g. `*_e2e_test.go`).

## Naming

- Functions: `Test<Subject>_<Behavior>`.
- Subtests (`t.Run`): `ok/...` and `err/...`.
- Table-driven helpers: `cases`, `tc`, `want`, `got`, `wantErr`.

## Style

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
