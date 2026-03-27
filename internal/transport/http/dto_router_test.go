package http

import (
	"casino-transaction-system/internal/domain"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTransactionRequest_ToDomain(t *testing.T) {
	cases := []struct {
		name       string
		req        CreateTransactionRequest
		wantAmount int64
		wantZero   bool
	}{
		{
			name: "ok/parses_valid_timestamp",
			req: CreateTransactionRequest{
				UserID:          11,
				TransactionType: domain.TransactionTypeWin,
				Amount:          "7710",
				Timestamp:       "2026-03-27T10:20:30Z",
			},
			wantAmount: 7710,
			wantZero:   false,
		},
		{
			name: "err/invalid_timestamp_returns_zero_time",
			req: CreateTransactionRequest{
				UserID:          1,
				TransactionType: domain.TransactionTypeBet,
				Amount:          "110",
				Timestamp:       "not-a-time",
			},
			wantAmount: 110,
			wantZero:   true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.req.ToDomain()
			if err != nil {
				t.Fatalf("ToDomain() error = %v", err)
			}
			if got.UserID != tc.req.UserID || got.Type != tc.req.TransactionType {
				t.Fatalf("ToDomain() mapped unexpected fields: %+v", got)
			}
			if got.Amount != tc.wantAmount {
				t.Fatalf("ToDomain() amount = %d, want %d", got.Amount, tc.wantAmount)
			}
			if got.Timestamp.IsZero() != tc.wantZero {
				t.Fatalf("ToDomain() timestamp zero = %v, want %v", got.Timestamp.IsZero(), tc.wantZero)
			}
		})
	}
}

func TestCreateTransactionRequest_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    CreateTransactionRequest
		wantErr bool
	}{
		{
			name:    "ok/amount_as_string",
			payload: `{"user_id":1,"transaction_type":"bet","amount":"123.45","timestamp":"2026-03-27T10:20:30Z"}`,
			want: CreateTransactionRequest{
				UserID:          1,
				TransactionType: domain.TransactionTypeBet,
				Amount:          "123.45",
				Timestamp:       "2026-03-27T10:20:30Z",
			},
		},
		{
			name:    "ok/amount_as_number",
			payload: `{"user_id":2,"transaction_type":"win","amount":77.1,"timestamp":"2026-03-27T10:20:30Z"}`,
			want: CreateTransactionRequest{
				UserID:          2,
				TransactionType: domain.TransactionTypeWin,
				Amount:          "77.1",
				Timestamp:       "2026-03-27T10:20:30Z",
			},
		},
		{
			name:    "err/missing_amount",
			payload: `{"user_id":3,"transaction_type":"win","timestamp":"2026-03-27T10:20:30Z"}`,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var got CreateTransactionRequest
			err := json.Unmarshal([]byte(tc.payload), &got)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if got.UserID != tc.want.UserID || got.TransactionType != tc.want.TransactionType || got.Amount != tc.want.Amount || got.Timestamp != tc.want.Timestamp {
				t.Fatalf("Unmarshal() got = %+v, want %+v", got, tc.want)
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
