package domain

import (
	"testing"
)

func TestTransaction_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tr      Transaction
		wantErr error
	}{
		{
			name: "valid bet",
			tr: Transaction{
				UserID: 1,
				Type:   TransactionTypeBet,
				Amount: 10.5,
			},
			wantErr: nil,
		},
		{
			name: "valid win",
			tr: Transaction{
				UserID: 1,
				Type:   TransactionTypeWin,
				Amount: 50,
			},
			wantErr: nil,
		},
		{
			name: "invalid user id",
			tr: Transaction{
				UserID: 0,
				Type:   TransactionTypeBet,
				Amount: 10.5,
			},
			wantErr: ErrInvalidUserID,
		},
		{
			name: "invalid amount",
			tr: Transaction{
				UserID: 1,
				Type:   TransactionTypeBet,
				Amount: -5,
			},
			wantErr: ErrInvalidAmount,
		},
		{
			name: "zero amount",
			tr: Transaction{
				UserID: 1,
				Type:   TransactionTypeBet,
				Amount: 0,
			},
			wantErr: ErrInvalidAmount,
		},
		{
			name: "invalid transaction type",
			tr: Transaction{
				UserID: 1,
				Type:   TransactionType("bonus"),
				Amount: 10,
			},
			wantErr: ErrInvalidTransactionType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tr.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
