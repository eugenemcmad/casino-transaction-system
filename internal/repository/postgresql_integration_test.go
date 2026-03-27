//go:build integration

package repository

import (
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/internal/testutil"
	"context"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestPostgresRepo_IntegrationFlow provides isolated testing for the data access layer.
// It uses testcontainers to ensure a clean, real PostgreSQL environment.
func TestPostgresRepo_IntegrationFlow(t *testing.T) {
	// 1. Setup isolated database
	connStr, cleanup := testutil.SetupPostgres(t)
	defer cleanup()

	repo, err := NewPostgresRepo(connStr)
	if err != nil {
		t.Fatalf("NewPostgresRepo() error = %v", err)
	}
	ctx := context.Background()

	t.Run("ok/save_is_idempotent", func(t *testing.T) {
		tr := domain.Transaction{
			UserID:    12345,
			Type:      domain.TransactionTypeBet,
			Amount:    9999,
			Timestamp: time.Now().UTC().Truncate(time.Microsecond),
		}

		// Save first time
		if err := repo.Save(ctx, tr); err != nil {
			t.Fatalf("failed initial save: %v", err)
		}

		// Save same data again (should be ignored by ON CONFLICT)
		if err := repo.Save(ctx, tr); err != nil {
			t.Fatalf("failed duplicate save: %v", err)
		}

		// Verify only one entry exists
		res, _ := repo.Get(ctx, 12345, nil)
		if len(res) != 1 {
			t.Errorf("idempotency check failed: expected 1 record, got %d", len(res))
		}
	})

	t.Run("ok/get_sorts_by_timestamp_desc", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		_ = repo.Save(ctx, domain.Transaction{UserID: 1, Type: "bet", Amount: 1000, Timestamp: now.Add(-time.Hour)})
		_ = repo.Save(ctx, domain.Transaction{UserID: 1, Type: "win", Amount: 2000, Timestamp: now})

		// Test retrieval and sorting (Default is DESC)
		res, _ := repo.Get(ctx, 1, nil)
		if len(res) != 2 {
			t.Errorf("expected 2 records, got %d", len(res))
		}
		if res[0].Amount != 2000 {
			t.Error("sorting check failed: latest transaction should be first")
		}
	})
}
