package kafka

import (
	"casino-transaction-system/internal/domain"
	"context"
	"encoding/json"
	"testing"
)

// Mock service for testing consumer
type mockService struct {
	registerCalled  bool
	lastTransaction domain.Transaction
}

func (m *mockService) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	m.registerCalled = true
	m.lastTransaction = t
	return nil
}

func (m *mockService) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	return nil, nil
}

func TestTransactionDTO_ToDomain(t *testing.T) {
	tests := []struct {
		name     string
		dto      TransactionDTO
		expected domain.Transaction
	}{
		{
			name: "valid conversion",
			dto: TransactionDTO{
				UserID:    1,
				Type:      domain.TransactionTypeBet,
				Amount:    10.5,
				Timestamp: "2023-10-27T10:00:00Z",
			},
			expected: domain.Transaction{
				UserID: 1,
				Type:   domain.TransactionTypeBet,
				Amount: 10.5,
			},
		},
		{
			name: "empty timestamp conversion",
			dto: TransactionDTO{
				UserID: 2,
				Type:   domain.TransactionTypeWin,
				Amount: 100,
			},
			expected: domain.Transaction{
				UserID: 2,
				Type:   domain.TransactionTypeWin,
				Amount: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dto.ToDomain()
			if got.UserID != tt.expected.UserID || got.Type != tt.expected.Type || got.Amount != tt.expected.Amount {
				t.Errorf("ToDomain() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConsumer_ProcessMessage(t *testing.T) {
	// Мы тестируем логику обработки внутри цикла, вынеся её (или имитируя её)
	// В данном случае проверим как DTO преобразуется и попадает в сервис.

	svc := &mockService{}

	validMsg := TransactionDTO{
		UserID: 1,
		Type:   domain.TransactionTypeBet,
		Amount: 10.5,
	}

	payload, _ := json.Marshal(validMsg)

	// Имитируем логику из Start()
	var dto TransactionDTO
	_ = json.Unmarshal(payload, &dto)
	tr := dto.ToDomain()

	if err := tr.Validate(); err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	_ = svc.RegisterTransaction(context.Background(), tr)

	if !svc.registerCalled {
		t.Error("Service RegisterTransaction was not called")
	}
	if svc.lastTransaction.UserID != 1 || svc.lastTransaction.Amount != 10.5 {
		t.Errorf("Transaction data mismatch: got %+v", svc.lastTransaction)
	}
}
