// Package service implements application use cases for transactions.
package service

import (
	"casino-transaction-system/internal/domain"
	"context"
)

// TransactionService defines the business logic for transactions (use cases).
type TransactionService interface {
	// RegisterTransaction persists a new transaction (e.g. from Kafka or API).
	RegisterTransaction(ctx context.Context, t domain.Transaction) error
	// RegisterTransactions efficiently persists multiple transactions.
	RegisterTransactions(ctx context.Context, txs []domain.Transaction) error
	// GetTransactions returns history filtered by optional user and type (userID 0 means all users).
	GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error)
}
