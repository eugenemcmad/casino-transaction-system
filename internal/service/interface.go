package service

import (
	"casino-transaction-system/internal/domain"
	"context"
)

// TransactionService defines the business logic for transactions.
type TransactionService interface {
	// RegisterTransaction processes and records a new bet or win.
	RegisterTransaction(ctx context.Context, t domain.Transaction) error
	GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error)
}

// TransactionRepository defines the data storage requirements.
type TransactionRepository interface {
	Save(ctx context.Context, t domain.Transaction) error
	GetByUserID(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error)
}
