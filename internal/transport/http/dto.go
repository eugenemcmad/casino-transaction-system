package http

import (
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/pkg/timeutil"
	"time"
)

type CreateTransactionRequest struct {
	UserID          int64                  `json:"user_id"`
	TransactionType domain.TransactionType `json:"transaction_type"`
	Amount          float64                `json:"amount"` 
	Timestamp       string                 `json:"timestamp"` // Standard string for flexible parsing
}

func (r CreateTransactionRequest) ToDomain() domain.Transaction {
	parsedTime, _ := timeutil.Parse(r.Timestamp)

	return domain.Transaction{
		UserID:    r.UserID,
		Type:      r.TransactionType,
		Amount:    r.Amount,
		Timestamp: parsedTime,
	}
}

type TransactionResponse struct {
	UserID          int64                  `json:"user_id"`
	TransactionType domain.TransactionType `json:"transaction_type"`
	Amount          float64                `json:"amount"`
	Timestamp       time.Time              `json:"timestamp"`
}

func NewTransactionResponse(t domain.Transaction) TransactionResponse {
	return TransactionResponse{
		UserID:          t.UserID,
		TransactionType: t.Type,
		Amount:          t.Amount,
		Timestamp:       t.Timestamp,
	}
}
