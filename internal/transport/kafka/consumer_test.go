package kafka

import (
	"casino-transaction-system/internal/domain"
	"testing"
)

func TestTransactionDTO_ToDomain(t *testing.T) {
	tests := []struct {
		name string
		dto  TransactionDTO
		want domain.Transaction
	}{
		{
			name: "Valid Conversion",
			dto: TransactionDTO{
				UserID:    1,
				Type:      "bet",
				Amount:    10.5,
				Timestamp: "2023-10-27T15:00:00Z",
			},
			want: domain.Transaction{
				UserID: 1,
				Type:   "bet",
				Amount: 10.5,
			},
		},
		{
			name: "Invalid Timestamp - returns zero time",
			dto: TransactionDTO{
				UserID:    1,
				Type:      "win",
				Amount:    100,
				Timestamp: "invalid-date",
			},
			want: domain.Transaction{
				UserID: 1,
				Type:   "win",
				Amount: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dto.ToDomain()
			if got.UserID != tt.want.UserID || got.Type != tt.want.Type || got.Amount != tt.want.Amount {
				t.Errorf("ToDomain() = %+v, want %+v", got, tt.want)
			}
			if tt.dto.Timestamp == "invalid-date" && !got.Timestamp.IsZero() {
				t.Error("Expected zero timestamp for invalid input")
			}
		})
	}
}
