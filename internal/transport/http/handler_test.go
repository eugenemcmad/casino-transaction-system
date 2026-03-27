package http

import (
	"casino-transaction-system/internal/domain"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestTransactionHandler_GetTransactions_Detailed(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		setupMock      func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error)
		expectedStatus int
	}{
		{
			name: "Success - all params",
			url:  "/transactions?user_id=1&transaction_type=bet",
			setupMock: func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
				return func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
					return []domain.Transaction{{UserID: 1, Type: "bet", Amount: 10}}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Error - invalid user_id",
			url:            "/transactions?user_id=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Error - invalid transaction_type",
			url:            "/transactions?transaction_type=invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Error - service failure",
			url:  "/transactions",
			setupMock: func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
				return func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
					return nil, errors.New("service error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockService{}
			if tt.setupMock != nil {
				svc.getTransactionsFunc = tt.setupMock()
			}
			h := NewTransactionHandler(svc)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			h.GetTransactions(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("GetTransactions() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestNewTransactionResponse(t *testing.T) {
	domainTx := domain.Transaction{
		UserID: 1,
		Type:   "bet",
		Amount: 10.5,
	}
	resp := NewTransactionResponse(domainTx)
	if resp.UserID != 1 || resp.Amount != 10.5 || resp.TransactionType != "bet" {
		t.Errorf("Mapping failed: %+v", resp)
	}
}
