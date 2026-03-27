package http

import (
	"casino-transaction-system/internal/domain"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTransactionRequest_ToDomain(t *testing.T) {
	cases := []struct {
		name     string
		req      CreateTransactionRequest
		wantZero bool
	}{
		{
			name: "ok/parses_valid_timestamp",
			req: CreateTransactionRequest{
				UserID:          11,
				TransactionType: domain.TransactionTypeWin,
				Amount:          77.1,
				Timestamp:       "2026-03-27T10:20:30Z",
			},
			wantZero: false,
		},
		{
			name: "err/invalid_timestamp_returns_zero_time",
			req: CreateTransactionRequest{
				UserID:          1,
				TransactionType: domain.TransactionTypeBet,
				Amount:          1.1,
				Timestamp:       "not-a-time",
			},
			wantZero: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := tc.req.ToDomain()
			if got.UserID != tc.req.UserID || got.Type != tc.req.TransactionType || got.Amount != tc.req.Amount {
				t.Fatalf("ToDomain() mapped unexpected fields: %+v", got)
			}
			if got.Timestamp.IsZero() != tc.wantZero {
				t.Fatalf("ToDomain() timestamp zero = %v, want %v", got.Timestamp.IsZero(), tc.wantZero)
			}
		})
	}
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
