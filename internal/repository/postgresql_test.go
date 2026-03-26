//go:build integration

package repository

import (
	"casino-transaction-system/internal/domain"
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testRepo    *PostgresRepo
	pgContainer *postgres.PostgresContainer
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// 1. Start PostgreSQL container via testcontainers
	fmt.Println("🚀 Starting PostgreSQL container for repository tests...")
	var err error
	pgContainer, err = postgres.RunContainer(ctx,
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
		fmt.Printf("❌ Failed to start PostgreSQL: %v\n", err)
		os.Exit(1)
	}

	pgConnStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")

	// 2. Setup Database Schema
	db, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		fmt.Printf("❌ Failed to open test db: %v\n", err)
		os.Exit(1)
	}
	_, err = db.Exec(`
		CREATE TABLE transactions (
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
		fmt.Printf("❌ Failed to setup schema: %v\n", err)
		os.Exit(1)
	}
	db.Close()

	// 3. Initialize repository
	testRepo = NewPostgresRepo(pgConnStr)

	code := m.Run()

	// 4. Teardown
	fmt.Println("🧹 Cleaning up PostgreSQL container...")
	pgContainer.Terminate(ctx)
	os.Exit(code)
}

// setupTest cleans the database before each test
func setupTest(t *testing.T) {
	_, err := testRepo.db.Exec("DELETE FROM transactions")
	if err != nil {
		t.Fatalf("Failed to clean transactions table: %v", err)
	}
}

func TestPostgresRepo_Save(t *testing.T) {
	ctx := context.Background()
	tr := domain.Transaction{
		UserID:    1,
		Type:      domain.TransactionTypeBet,
		Amount:    100.50,
		Timestamp: time.Now().UTC().Truncate(time.Microsecond),
	}

	t.Run("successful save", func(t *testing.T) {
		setupTest(t)
		err := testRepo.Save(ctx, tr)
		if err != nil {
			t.Errorf("Save() failed: %v", err)
		}
	})

	t.Run("idempotency check", func(t *testing.T) {
		setupTest(t)
		_ = testRepo.Save(ctx, tr)
		_ = testRepo.Save(ctx, tr) // Duplicate save

		res, _ := testRepo.Get(ctx, 1, nil) // Corrected to Get
		if len(res) != 1 {
			t.Errorf("Expected 1 record after duplicate Save, got %d", len(res))
		}
	})
}

func TestPostgresRepo_Get(t *testing.T) { // Corrected name
	setupTest(t)
	ctx := context.Background()

	_ = testRepo.Save(ctx, domain.Transaction{UserID: 1, Type: "bet", Amount: 10, Timestamp: time.Now()})
	_ = testRepo.Save(ctx, domain.Transaction{UserID: 2, Type: "win", Amount: 20, Timestamp: time.Now()})

	tests := []struct {
		name   string
		userID int64
		want   int
	}{
		{"find user 1", 1, 1},
		{"find all (userID=0)", 0, 2},
		{"non-existent user", 999, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := testRepo.Get(ctx, tt.userID, nil) // Corrected to Get
			if err != nil {
				t.Errorf("Get() error = %v", err)
			}
			if len(res) != tt.want {
				t.Errorf("Got %d, want %d", len(res), tt.want)
			}
		})
	}
}
