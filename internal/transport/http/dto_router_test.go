package http

import (
	"casino-transaction-system/internal/domain"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateTransactionRequest_ToDomain(t *testing.T) {
	t.Run("ok/parses_valid_timestamp", func(t *testing.T) {
		req := CreateTransactionRequest{
			UserID:          11,
			TransactionType: domain.TransactionTypeWin,
			Amount:          77.1,
			Timestamp:       "2026-03-27T10:20:30Z",
		}

		got := req.ToDomain()
		if got.UserID != req.UserID || got.Type != req.TransactionType || got.Amount != req.Amount {
			t.Fatalf("ToDomain() mapped unexpected fields: %+v", got)
		}
		if got.Timestamp.IsZero() {
			t.Fatal("ToDomain() expected parsed timestamp, got zero value")
		}
	})

	t.Run("err/invalid_timestamp_returns_zero_time", func(t *testing.T) {
		req := CreateTransactionRequest{
			UserID:          1,
			TransactionType: domain.TransactionTypeBet,
			Amount:          1.1,
			Timestamp:       "not-a-time",
		}

		got := req.ToDomain()
		if !got.Timestamp.Equal(time.Time{}) {
			t.Fatalf("ToDomain() expected zero timestamp, got %v", got.Timestamp)
		}
	})
}

func TestNewRouter_RegistersHealthRoute(t *testing.T) {
	handler := NewTransactionHandler(&mockService{
		getTransactionsFunc: func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
			return []domain.Transaction{}, nil
		},
	})
	router := NewRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", w.Code, http.StatusOK)
	}
	if body := w.Body.String(); body != "OK" {
		t.Fatalf("health body = %q, want %q", body, "OK")
	}
}
