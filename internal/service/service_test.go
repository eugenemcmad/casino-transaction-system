package service

import (
	"casino-transaction-system/internal/domain"
	"context"
	"errors"
	"testing"
)

// mockRepo - простая реализация мока для тестов
type mockRepo struct {
	saveCalled bool
	getFunc    func() ([]domain.Transaction, error)
}

func (m *mockRepo) Save(ctx context.Context, t domain.Transaction) error {
	m.saveCalled = true
	return nil
}

// Get implements domain.TransactionRepository (renamed from Find)
func (m *mockRepo) Get(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	if m.getFunc != nil {
		return m.getFunc()
	}
	return nil, nil
}

func TestTransactionService_RegisterTransaction(t *testing.T) {
	repo := &mockRepo{}
	svc := NewTransactionService(repo)

	tr := domain.Transaction{UserID: 1, Type: domain.TransactionTypeBet, Amount: 10}
	err := svc.RegisterTransaction(context.Background(), tr)

	if err != nil {
		t.Errorf("RegisterTransaction() unexpected error = %v", err)
	}
	if !repo.saveCalled {
		t.Error("RegisterTransaction() did not call repository Save")
	}
}

func TestTransactionService_GetTransactions(t *testing.T) {
	expected := []domain.Transaction{
		{UserID: 1, Amount: 100},
		{UserID: 1, Amount: 200},
	}

	repo := &mockRepo{
		getFunc: func() ([]domain.Transaction, error) {
			return expected, nil
		},
	}
	svc := NewTransactionService(repo)

	got, err := svc.GetTransactions(context.Background(), 1, nil)

	if err != nil {
		t.Errorf("GetTransactions() unexpected error = %v", err)
	}
	if len(got) != len(expected) {
		t.Errorf("GetTransactions() got %d items, want %d", len(got), len(expected))
	}

	// Тест ошибки репозитория
	repoErr := errors.New("db error")
	repo.getFunc = func() ([]domain.Transaction, error) {
		return nil, repoErr
	}

	_, err = svc.GetTransactions(context.Background(), 1, nil)
	if !errors.Is(err, repoErr) {
		t.Errorf("GetTransactions() error = %v, want %v", err, repoErr)
	}
}
