package domain

import "time"

// Transaction is the core domain model.
type Transaction struct {
	ID        int64
	UserID    int64
	Type      TransactionType
	Amount    float64 // But in a production system, it's better to use int64 for cents
	Timestamp time.Time
	CreatedAt time.Time
}
