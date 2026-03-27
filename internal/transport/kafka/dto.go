package kafka

import (
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/pkg/money"
	"casino-transaction-system/pkg/timeutil"
	"encoding/json"
	"fmt"
	"log/slog"
)

type TransactionDTO struct {
	UserID    int64                  `json:"user_id"`
	Type      domain.TransactionType `json:"transaction_type"`
	Amount    string                 `json:"amount"`
	Timestamp string                 `json:"timestamp"`
}

func (dto *TransactionDTO) UnmarshalJSON(data []byte) error {
	var aux struct {
		UserID    int64                  `json:"user_id"`
		Type      domain.TransactionType `json:"transaction_type"`
		Amount    json.RawMessage        `json:"amount"`
		Timestamp string                 `json:"timestamp"`
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

	dto.UserID = aux.UserID
	dto.Type = aux.Type
	dto.Amount = amountStr
	dto.Timestamp = aux.Timestamp
	return nil
}

func (dto TransactionDTO) ToDomain() (domain.Transaction, error) {
	parsedTime, err := timeutil.Parse(dto.Timestamp)
	if err != nil && dto.Timestamp != "" {
		slog.Warn("failed to parse timestamp from Kafka", "timestamp_raw", dto.Timestamp, "error", err)
	}
	amount, err := money.ParseToMinorUnits(dto.Amount)
	if err != nil {
		return domain.Transaction{}, err
	}

	return domain.Transaction{
		UserID:    dto.UserID,
		Type:      dto.Type,
		Amount:    amount,
		Timestamp: parsedTime,
	}, nil
}
