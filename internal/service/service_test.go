package service

import (
	"casino-transaction-system/internal/domain"
	"context"
	"errors"
	"testing"
)

type mockRepository struct {
	saveCalled bool
	saveErr    error
	getFunc    func() ([]domain.Transaction, error)
}

func (m *mockRepository) Save(ctx context.Context, t domain.Transaction) error {
	m.saveCalled = true
	return m.saveErr
}

func (m *mockRepository) Get(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	if m.getFunc != nil {
		return m.getFunc()
	}
	return nil, nil
}

func TestTransactionService_RegisterTransaction_CallsRepositorySave(t *testing.T) {
	cases := []struct {
		name    string
		saveErr error
		wantErr bool
	}{
		{name: "ok/calls_repository_save", saveErr: nil, wantErr: false},
		{name: "err/returns_repository_error", saveErr: errors.New("save failed"), wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepository{saveErr: tc.saveErr}
			svc := NewTransactionService(repo)
			tx := domain.Transaction{UserID: 1, Type: domain.TransactionTypeBet, Amount: 10}

			err := svc.RegisterTransaction(context.Background(), tx)
			if (err != nil) != tc.wantErr {
				t.Fatalf("RegisterTransaction() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !repo.saveCalled {
				t.Fatal("RegisterTransaction() did not call repository Save")
			}
		})
	}
}

func TestTransactionService_GetTransactions_ReturnsRepositoryResults(t *testing.T) {
	repoErr := errors.New("db error")
	want := []domain.Transaction{
		{UserID: 1, Amount: 100},
		{UserID: 1, Amount: 200},
	}
	cases := []struct {
		name    string
		getFunc func() ([]domain.Transaction, error)
		wantLen int
		wantErr error
	}{
		{
			name: "ok/returns_repository_results",
			getFunc: func() ([]domain.Transaction, error) {
				return want, nil
			},
			wantLen: len(want),
		},
		{
			name: "err/returns_repository_error",
			getFunc: func() ([]domain.Transaction, error) {
				return nil, repoErr
			},
			wantErr: repoErr,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepository{getFunc: tc.getFunc}
			svc := NewTransactionService(repo)

			got, err := svc.GetTransactions(context.Background(), 1, nil)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("GetTransactions() error = %v, want %v", err, tc.wantErr)
			}
			if len(got) != tc.wantLen {
				t.Fatalf("GetTransactions() len = %d, want %d", len(got), tc.wantLen)
			}
		})
	}
}
