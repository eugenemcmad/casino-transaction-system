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

func TestTransactionHandler_GetTransactions_ReturnsExpectedStatusCodes(t *testing.T) {
	cases := []struct {
		name       string
		url        string
		setupMock  func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error)
		wantStatus int
	}{
		{
			name: "ok/returns_transactions_for_all_params",
			url:  "/transactions?user_id=1&transaction_type=bet",
			setupMock: func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
				return func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
					return []domain.Transaction{{UserID: 1, Type: "bet", Amount: 10}}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "err/invalid_user_id",
			url:        "/transactions?user_id=abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "err/non_positive_user_id_zero",
			url:        "/transactions?user_id=0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "err/non_positive_user_id_negative",
			url:        "/transactions?user_id=-1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "err/invalid_transaction_type",
			url:        "/transactions?transaction_type=invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "err/service_failure",
			url:  "/transactions",
			setupMock: func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
				return func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
					return nil, errors.New("service error")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			if tc.setupMock != nil {
				svc.getTransactionsFunc = tc.setupMock()
			}
			h := NewTransactionHandler(svc)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			w := httptest.NewRecorder()

			h.GetTransactions(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("GetTransactions() status = %v, want %v", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestNewTransactionResponse_MapsDomainFields(t *testing.T) {
	domainTx := domain.Transaction{
		UserID: 1,
		Type:   "bet",
		Amount: 1050,
	}
	resp := NewTransactionResponse(domainTx)
	if resp.UserID != 1 || resp.Amount != 1050 || resp.TransactionType != "bet" {
		t.Errorf("Mapping failed: %+v", resp)
	}
}
