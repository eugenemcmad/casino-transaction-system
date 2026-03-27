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
		res, err := repo.Get(ctx, 12345, nil)
		if err != nil {
			t.Fatalf("repo.Get() error = %v", err)
		}
		wantLen := 1
		if len(res) != wantLen {
			t.Errorf("len(res) = %d, want %d", len(res), wantLen)
		}
	})

	t.Run("ok/get_sorts_by_timestamp_desc", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		if err := repo.Save(ctx, domain.Transaction{UserID: 1, Type: "bet", Amount: 1000, Timestamp: now.Add(-time.Hour)}); err != nil {
			t.Fatalf("repo.Save() error = %v", err)
		}
		if err := repo.Save(ctx, domain.Transaction{UserID: 1, Type: "win", Amount: 2000, Timestamp: now}); err != nil {
			t.Fatalf("repo.Save() error = %v", err)
		}

		// Test retrieval and sorting (Default is DESC)
		res, err := repo.Get(ctx, 1, nil)
		if err != nil {
			t.Fatalf("repo.Get() error = %v", err)
		}
		wantLen := 2
		if len(res) != wantLen {
			t.Errorf("len(res) = %d, want %d", len(res), wantLen)
		}
		wantFirstAmount := int64(2000)
		if len(res) > 0 && res[0].Amount != wantFirstAmount {
			t.Errorf("res[0].Amount = %d, want %d (latest first)", res[0].Amount, wantFirstAmount)
		}
	})
}
