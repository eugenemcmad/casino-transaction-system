package domain

import (
	"time"
)

// Transaction is the core domain model. It contains no JSON/DB tags
// and encapsulates its own business validation rules (Rich Domain Model).
type Transaction struct {
	ID        int64
	UserID    int64
	Type      TransactionType
	Amount    float64
	Timestamp time.Time
	CreatedAt time.Time
}

// Validate ensures the transaction follows business rules.
func (t *Transaction) Validate() error {
	if t.UserID <= 0 {
		return ErrInvalidUserID
	}
	if t.Amount <= 0 {
		return ErrInvalidAmount
	}
	if err := t.Type.IsValid(); err != nil {
		return err
	}
	return nil
}
