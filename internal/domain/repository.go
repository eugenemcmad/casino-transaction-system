package domain

import (
	"context"
)

// TransactionRepository defines the persistence port for transactions.
type TransactionRepository interface {
	// Save inserts a transaction; implementations may treat duplicates as no-ops.
	Save(ctx context.Context, t Transaction) error
	// SaveBulk inserts multiple transactions efficiently.
	SaveBulk(ctx context.Context, txs []Transaction) error
	// Get returns transactions filtered by optional user and type (userID 0 means all users).
	Get(ctx context.Context, userID int64, tType *TransactionType) ([]Transaction, error)
}
