//go:build integration || test || e2e

package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupPostgres starts a temporary Postgres container and creates the schema.
// Returns connection string and a cleanup function.
func SetupPostgres(t *testing.T) (string, func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Skipf("skipping postgres testcontainer setup (docker unavailable): %v", r)
		}
	}()
	ctx := context.Background()

	t.Log("spinning up temporary PostgreSQL container")
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(2*time.Minute)),
	)
	if err != nil {
		t.Skipf("skipping postgres testcontainer setup (docker unavailable): %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Apply schema manually for integration tests to ensure clean state
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS transactions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			type VARCHAR(10) NOT NULL,
			amount NUMERIC(15, 2) NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			UNIQUE NULLS NOT DISTINCT (user_id, type, amount, timestamp)
		);
	`)
	if err != nil {
		t.Fatalf("failed to setup schema: %v", err)
	}

	t.Log("PostgreSQL container is ready and schema applied")

	return connStr, func() {
		pgContainer.Terminate(ctx)
	}
}
