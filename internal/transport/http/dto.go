package http

import (
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/pkg/money"
	"casino-transaction-system/pkg/timeutil"
	"encoding/json"
	"fmt"
	"time"
)

type CreateTransactionRequest struct {
	UserID          int64                  `json:"user_id"`
	TransactionType domain.TransactionType `json:"transaction_type"`
	Amount          string                 `json:"amount"`
	Timestamp       string                 `json:"timestamp"` // Standard string for flexible parsing
}

func (r *CreateTransactionRequest) UnmarshalJSON(data []byte) error {
	var aux struct {
		UserID          int64                  `json:"user_id"`
		TransactionType domain.TransactionType `json:"transaction_type"`
		Amount          json.RawMessage        `json:"amount"`
		Timestamp       string                 `json:"timestamp"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var amountStr string
	if len(aux.Amount) == 0 {
		return fmt.Errorf("amount is required")
	}

	if aux.Amount[0] == '"' {
		if err := json.Unmarshal(aux.Amount, &amountStr); err != nil {
			return err
		}
	} else {
		var num json.Number
		if err := json.Unmarshal(aux.Amount, &num); err != nil {
			return err
		}
		amountStr = num.String()
	}

	r.UserID = aux.UserID
	r.TransactionType = aux.TransactionType
	r.Amount = amountStr
	r.Timestamp = aux.Timestamp
	return nil
}

func (r CreateTransactionRequest) ToDomain() (domain.Transaction, error) {
	parsedTime, _ := timeutil.Parse(r.Timestamp)
	amount, err := money.ParseToMinorUnits(r.Amount)
	if err != nil {
		return domain.Transaction{}, err
	}

	return domain.Transaction{
		UserID:    r.UserID,
		Type:      r.TransactionType,
		Amount:    amount,
		Timestamp: parsedTime,
	}, nil
}

type TransactionResponse struct {
	UserID          int64                  `json:"user_id"`
	TransactionType domain.TransactionType `json:"transaction_type"`
	Amount          int64                  `json:"amount"`
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
