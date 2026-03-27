//go:build integration

package http

import (
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/internal/repository"
	"casino-transaction-system/internal/service"
	"casino-transaction-system/internal/testutil"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/lib/pq"
)

// TestAPI_IntegrationFlow verifies HTTP handlers using a real database managed by testcontainers.
func TestAPI_IntegrationFlow(t *testing.T) {
	// 1. Initialize temporary Postgres via testutil
	connStr, cleanup := testutil.SetupPostgres(t)
	defer cleanup()

	repo, err := repository.NewPostgresRepo(connStr)
	if err != nil {
		t.Fatalf("NewPostgresRepo() error = %v", err)
	}
	svc := service.NewTransactionService(repo)
	handler := NewTransactionHandler(svc)
	ctx := context.Background()

	// Direct DB connection for test state management (cleanup and seeding)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to connect to test db directly: %v", err)
	}
	defer db.Close()

	t.Run("ok/filters_by_user_id", func(t *testing.T) {
		// Clear table before run
		_, _ = db.Exec("DELETE FROM transactions")

		uid := int64(2024)
		_ = repo.Save(ctx, domain.Transaction{UserID: uid, Type: domain.TransactionTypeWin, Amount: 500})

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions?user_id=%d", uid), nil)
		w := httptest.NewRecorder()
		handler.GetTransactions(w, req)

		var resp []TransactionResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}
		if len(resp) != 1 || resp[0].UserID != uid {
			t.Errorf("expected 1 record for user %d, got %d", uid, len(resp))
		}
	})

	t.Run("ok/returns_all_users_when_no_filters", func(t *testing.T) {
		_, _ = db.Exec("DELETE FROM transactions")
		repo.Save(ctx, domain.Transaction{UserID: 1, Type: "bet", Amount: 10})
		repo.Save(ctx, domain.Transaction{UserID: 2, Type: "win", Amount: 20})

		req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
		w := httptest.NewRecorder()
		handler.GetTransactions(w, req)

		var resp []TransactionResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(resp) != 2 {
			t.Errorf("expected total of 2 records, got %d", len(resp))
		}
	})
}
