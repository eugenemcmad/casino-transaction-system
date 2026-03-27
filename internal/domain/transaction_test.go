package domain

import (
	"testing"
)

func TestTransaction_Validate(t *testing.T) {
	tests := []struct {
		name    string
		tx      Transaction
		wantErr bool
	}{
		{
			name: "Valid Bet",
			tx: Transaction{
				UserID: 1,
				Type:   TransactionTypeBet,
				Amount: 10.5,
			},
			wantErr: false,
		},
		{
			name: "Valid Win",
			tx: Transaction{
				UserID: 1,
				Type:   TransactionTypeWin,
				Amount: 100,
			},
			wantErr: false,
		},
		{
			name: "Invalid UserID (Zero)",
			tx: Transaction{
				UserID: 0,
				Type:   TransactionTypeBet,
				Amount: 10,
			},
			wantErr: true,
		},
		{
			name: "Invalid Amount (Zero)",
			tx: Transaction{
				UserID: 1,
				Type:   TransactionTypeBet,
				Amount: 0,
			},
			wantErr: true,
		},
		{
			name: "Invalid Amount (Negative)",
			tx: Transaction{
				UserID: 1,
				Type:   TransactionTypeBet,
				Amount: -5,
			},
			wantErr: true,
		},
		{
			name: "Invalid Type",
			tx: Transaction{
				UserID: 1,
				Type:   "bonus",
				Amount: 10,
			},
			wantErr: true,
		},
		{
			name: "Empty Type",
			tx: Transaction{
				UserID: 1,
				Type:   "",
				Amount: 10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tx.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Transaction.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransactionType_IsValid(t *testing.T) {
	if err := TransactionTypeBet.IsValid(); err != nil {
		t.Errorf("Bet should be valid, got error: %v", err)
	}
	if err := TransactionTypeWin.IsValid(); err != nil {
		t.Errorf("Win should be valid, got error: %v", err)
	}
	var invalid TransactionType = "invalid"
	if err := invalid.IsValid(); err == nil {
		t.Error("Invalid type should return error")
	}
}
