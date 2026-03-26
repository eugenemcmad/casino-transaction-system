package http

import (
	"casino-transaction-system/internal/domain"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock service for testing handler
type mockService struct {
	getTransactionsFunc func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error)
}

func (m *mockService) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	return nil
}

func (m *mockService) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	if m.getTransactionsFunc != nil {
		return m.getTransactionsFunc(ctx, userID, tType)
	}
	return nil, nil
}

func TestTransactionHandler_GetTransactions(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockData       []domain.Transaction
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "Success - single user transactions",
			url:  "/transactions?user_id=1",
			mockData: []domain.Transaction{
				{UserID: 1, Amount: 100, Type: domain.TransactionTypeBet},
				{UserID: 1, Amount: 50, Type: domain.TransactionTypeWin},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "Success - all users transactions (no user_id)",
			url:  "/transactions",
			mockData: []domain.Transaction{
				{UserID: 1, Amount: 100, Type: domain.TransactionTypeBet},
				{UserID: 2, Amount: 300, Type: domain.TransactionTypeWin},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "Success - with type filter",
			url:  "/transactions?user_id=1&transaction_type=bet",
			mockData: []domain.Transaction{
				{UserID: 1, Amount: 100, Type: domain.TransactionTypeBet},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "Error - invalid user_id format",
			url:            "/transactions?user_id=invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Error - invalid transaction type",
			url:            "/transactions?user_id=1&transaction_type=bonus",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockService{
				getTransactionsFunc: func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
					return tt.mockData, nil
				},
			}
			h := NewTransactionHandler(svc)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			h.GetTransactions(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("GetTransactions() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp []TransactionResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if len(resp) != tt.expectedCount {
					t.Errorf("GetTransactions() got %d items, want %d", len(resp), tt.expectedCount)
				}
			}
		})
	}
}
