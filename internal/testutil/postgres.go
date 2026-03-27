//go:build integration || test

package testutil

import (
	"context"
	"database/sql"
	"fmt"
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
	ctx := context.Background()

	fmt.Println("🐳 Spinning up temporary PostgreSQL container...")
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
		t.Fatalf("failed to start postgres: %v", err)
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

	fmt.Println("✅ PostgreSQL container is ready and schema applied.")

	return connStr, func() {
		pgContainer.Terminate(ctx)
	}
}
