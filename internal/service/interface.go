package service

import (
	"casino-transaction-system/internal/domain"
	"context"
)

// TransactionService defines the business logic for transactions (Use Case).
type TransactionService interface {
	RegisterTransaction(ctx context.Context, t domain.Transaction) error
	GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error)
}
