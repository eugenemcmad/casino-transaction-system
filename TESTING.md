# Testing Guidelines

How to **run** tests (commands, Docker, coverage): [`DEVELOPMENT.md`](DEVELOPMENT.md#test).

## Test Types

- **Unit tests**: default tests, no build tags.
- **Integration tests**: use `//go:build integration` and `*_integration_test.go`.
- **E2E tests**: use `//go:build e2e` and `*_e2e_test.go` under `e2e/`.

## Naming

- Test functions: `Test<Subject>_<Behavior>`.
- Subtests (`t.Run`): use `ok/...` and `err/...`.
- Table-driven variables: prefer `cases`, `tc`, `want`, `got`, `wantErr`.

## Test Style

- Use **table-driven tests** for multiple input/output combinations.
- Use **scenario-style tests** for heavy setup (e.g., `sqlmock`, containers, full flow checks).
- Do not mix unrelated concerns in one test function.

## Assertions

- Keep assertions explicit and focused.
- Compare behavior, not implementation details.
- Prefer clear failure messages with expected vs actual values.

## Logging

- In tests, use `t.Log` / `t.Logf` (not `fmt.Println`).
- Avoid emoji in test output.

## Execution

See [`DEVELOPMENT.md#test`](DEVELOPMENT.md#test) for commands and Makefile targets.
