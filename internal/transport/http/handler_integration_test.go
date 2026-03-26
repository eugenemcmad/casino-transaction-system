//go:build integration

package http

import (
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/internal/repository"
	"casino-transaction-system/internal/service"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testDBRepo  *repository.PostgresRepo
	pgContainer *postgres.PostgresContainer
	testConnStr string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// 1. Start Postgres
	fmt.Println("🚀 Starting PostgreSQL for API integration tests...")
	var err error
	pgContainer, err = postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	if err != nil {
		fmt.Printf("❌ Failed to start Postgres: %v\n", err)
		os.Exit(1)
	}

	testConnStr, _ = pgContainer.ConnectionString(ctx, "sslmode=disable")

	// 2. Setup Schema
	db, err := sql.Open("postgres", testConnStr)
	if err != nil {
		fmt.Printf("❌ Failed to open db: %v\n", err)
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

	testDBRepo = repository.NewPostgresRepo(testConnStr)

	code := m.Run()

	// 3. Teardown
	pgContainer.Terminate(ctx)
	os.Exit(code)
}

func setupTestDB(t *testing.T) {
	db, err := sql.Open("postgres", testConnStr)
	if err != nil {
		t.Fatalf("Failed to open db for cleanup: %v", err)
	}
	defer db.Close()
	_, _ = db.Exec("DELETE FROM transactions")
}

func TestAPI_GetTransactions_Integration(t *testing.T) {
	ctx := context.Background()
	setupTestDB(t)

	svc := service.NewTransactionService(testDBRepo)
	handler := NewTransactionHandler(svc)

	// Seed data
	u1 := int64(1001)
	u2 := int64(1002)

	_ = testDBRepo.Save(ctx, domain.Transaction{UserID: u1, Type: "bet", Amount: 50, Timestamp: time.Now()})
	_ = testDBRepo.Save(ctx, domain.Transaction{UserID: u1, Type: "win", Amount: 100, Timestamp: time.Now()})
	_ = testDBRepo.Save(ctx, domain.Transaction{UserID: u2, Type: "bet", Amount: 30, Timestamp: time.Now()})

	t.Run("Get transactions for specific user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions?user_id=%d", u1), nil)
		w := httptest.NewRecorder()

		handler.GetTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp []TransactionResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 2 {
			t.Errorf("expected 2 transactions, got %d", len(resp))
		}
	})

	t.Run("Get transactions for ALL users", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
		w := httptest.NewRecorder()

		handler.GetTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		var resp []TransactionResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 3 {
			t.Errorf("expected 3 transactions, got %d", len(resp))
		}
	})

	t.Run("Filter by type across all users", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/transactions?transaction_type=bet", nil)
		w := httptest.NewRecorder()

		handler.GetTransactions(w, req)

		var resp []TransactionResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 2 {
			t.Errorf("expected 2 bet transactions, got %d", len(resp))
		}
		for _, tx := range resp {
			if tx.TransactionType != domain.TransactionTypeBet {
				t.Errorf("Filter failed, found non-bet transaction: %+v", tx)
			}
		}
	})
}
