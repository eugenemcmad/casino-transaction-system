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
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Fatalf("failed to close test db directly: %v", closeErr)
		}
	}()

	t.Run("ok/filters_by_user_id", func(t *testing.T) {
		// Clear table before run
		if _, err := db.Exec("DELETE FROM transactions"); err != nil {
			t.Fatalf("db.Exec(delete) error = %v", err)
		}

		uid := int64(2024)
		if err := repo.Save(ctx, domain.Transaction{UserID: uid, Type: domain.TransactionTypeWin, Amount: 500}); err != nil {
			t.Fatalf("repo.Save() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/transactions?user_id=%d", uid), nil)
		w := httptest.NewRecorder()
		handler.GetTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GetTransactions() status = %d, want %d", w.Code, http.StatusOK)
		}

		var got []TransactionResponse
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("json.Decode() error = %v", err)
		}
		wantLen := 1
		if len(got) != wantLen {
			t.Fatalf("len(resp) = %d, want %d", len(got), wantLen)
		}
		if got[0].UserID != uid {
			t.Errorf("resp[0].UserID = %d, want %d", got[0].UserID, uid)
		}
	})

	t.Run("ok/returns_all_users_when_no_filters", func(t *testing.T) {
		if _, err := db.Exec("DELETE FROM transactions"); err != nil {
			t.Fatalf("db.Exec(delete) error = %v", err)
		}
		if err := repo.Save(ctx, domain.Transaction{UserID: 1, Type: "bet", Amount: 10}); err != nil {
			t.Fatalf("repo.Save() error = %v", err)
		}
		if err := repo.Save(ctx, domain.Transaction{UserID: 2, Type: "win", Amount: 20}); err != nil {
			t.Fatalf("repo.Save() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
		w := httptest.NewRecorder()
		handler.GetTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GetTransactions() status = %d, want %d", w.Code, http.StatusOK)
		}

		var got []TransactionResponse
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("json.Decode() error = %v", err)
		}
		wantLen := 2
		if len(got) != wantLen {
			t.Errorf("len(resp) = %d, want %d", len(got), wantLen)
		}
	})
}
