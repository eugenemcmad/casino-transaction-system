package domain

import (
	"errors"
	"testing"
)

func TestTransaction_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tx      Transaction
		wantErr error
	}{
		{
			name: "Valid Bet",
			tx:   Transaction{UserID: 1, Type: TransactionTypeBet, Amount: 10.5},
		},
		{
			name: "Valid Win",
			tx:   Transaction{UserID: 1, Type: TransactionTypeWin, Amount: 100},
		},
		{
			name:    "Invalid UserID",
			tx:      Transaction{UserID: 0, Type: TransactionTypeBet, Amount: 10},
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "Invalid Amount (zero)",
			tx:      Transaction{UserID: 1, Type: TransactionTypeBet, Amount: 0},
			wantErr: ErrInvalidAmount,
		},
		{
			name:    "Invalid Amount (negative)",
			tx:      Transaction{UserID: 1, Type: TransactionTypeBet, Amount: -10},
			wantErr: ErrInvalidAmount,
		},
		{
			name:    "Invalid Transaction Type",
			tx:      Transaction{UserID: 1, Type: "deposit", Amount: 100},
			wantErr: ErrInvalidTransactionType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransactionType_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		trType  TransactionType
		isValid bool
	}{
		{"Valid Bet", TransactionTypeBet, true},
		{"Valid Win", TransactionTypeWin, true},
		{"Invalid Type", "unknown", false},
		{"Empty Type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trType.IsValid()
			if (err == nil) != tt.isValid {
				t.Errorf("IsValid() for %s was %v, want %v", tt.trType, (err == nil), tt.isValid)
			}
		})
	}
}
