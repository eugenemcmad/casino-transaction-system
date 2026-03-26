package kafka

import (
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/pkg/timeutil"
	"log/slog"
)

type TransactionDTO struct {
	UserID    int64                  `json:"user_id"`
	Type      domain.TransactionType `json:"transaction_type"`
	Amount    float64                `json:"amount"` 
	Timestamp string                 `json:"timestamp"` // Standard string for flexible parsing
}

func (dto TransactionDTO) ToDomain() domain.Transaction {
	parsedTime, err := timeutil.Parse(dto.Timestamp)
	if err != nil && dto.Timestamp != "" {
		slog.Warn("failed to parse timestamp from Kafka", "timestamp_raw", dto.Timestamp, "error", err)
	}

	return domain.Transaction{
		UserID:    dto.UserID,
		Type:      dto.Type,
		Amount:    dto.Amount,
		Timestamp: parsedTime,
	}
}
