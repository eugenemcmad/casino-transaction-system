package domain

import (
	"context"
)

// TransactionRepository defines the data storage requirements (Domain Port).
type TransactionRepository interface {
	Save(ctx context.Context, t Transaction) error
	GetByUserID(ctx context.Context, userID int64, tType *TransactionType) ([]Transaction, error)
}
