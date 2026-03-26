package repository

import (
	"casino-transaction-system/internal/domain"
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) (*PostgresRepo, func()) {
	t.Helper()

	url := os.Getenv("TEST_PG_URL")
	if url == "" {
		url = "postgres://postgres:pass@127.0.0.1:5432/casino?sslmode=disable"
	}

	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Skip("Database not available, skipping integration test. Run 'make docker-up' first.")
	}

	// Clean table before test
	_, _ = db.Exec("DELETE FROM transactions")

	repo := &PostgresRepo{db: db}

	return repo, func() {
		db.Close()
	}
}

func TestPostgresRepo_Save(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	tr := domain.Transaction{
		UserID:    1,
		Type:      domain.TransactionTypeBet,
		Amount:    100.50,
		Timestamp: time.Now().UTC().Truncate(time.Microsecond),
	}

	t.Run("successful save", func(t *testing.T) {
		err := repo.Save(ctx, tr)
		if err != nil {
			t.Errorf("Save() failed: %v", err)
		}
	})

	t.Run("idempotency check", func(t *testing.T) {
		// Save same transaction again
		err := repo.Save(ctx, tr)
		if err != nil {
			t.Errorf("Save() failed on duplicate: %v", err)
		}

		// Verify only one exists
		res, _ := repo.GetByUserID(ctx, 1, nil)
		if len(res) != 1 {
			t.Errorf("Expected 1 record, got %d", len(res))
		}
	})
}

func TestPostgresRepo_GetByUserID(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	repo.Save(ctx, domain.Transaction{UserID: 1, Type: "bet", Amount: 10, Timestamp: time.Now()})
	repo.Save(ctx, domain.Transaction{UserID: 2, Type: "win", Amount: 20, Timestamp: time.Now()})

	tests := []struct {
		name   string
		userID int64
		want   int
	}{
		{"find user 1", 1, 1},
		{"find all", 0, 2},
		{"non-existent user", 999, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := repo.GetByUserID(ctx, tt.userID, nil)
			if err != nil {
				t.Errorf("GetByUserID() error = %v", err)
			}
			if len(res) != tt.want {
				t.Errorf("Got %d, want %d", len(res), tt.want)
			}
		})
	}
}
