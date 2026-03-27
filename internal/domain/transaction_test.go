package domain

import (
	"errors"
	"testing"
)

func TestTransaction_Validate(t *testing.T) {
	cases := []struct {
		name    string
		tx      Transaction
		wantErr error
	}{
		{
			name: "ok/valid_bet",
			tx:   Transaction{UserID: 1, Type: TransactionTypeBet, Amount: 10.5},
		},
		{
			name: "ok/valid_win",
			tx:   Transaction{UserID: 1, Type: TransactionTypeWin, Amount: 100},
		},
		{
			name:    "err/invalid_user_id",
			tx:      Transaction{UserID: 0, Type: TransactionTypeBet, Amount: 10},
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "err/invalid_amount_zero",
			tx:      Transaction{UserID: 1, Type: TransactionTypeBet, Amount: 0},
			wantErr: ErrInvalidAmount,
		},
		{
			name:    "err/invalid_amount_negative",
			tx:      Transaction{UserID: 1, Type: TransactionTypeBet, Amount: -10},
			wantErr: ErrInvalidAmount,
		},
		{
			name:    "err/invalid_transaction_type",
			tx:      Transaction{UserID: 1, Type: "deposit", Amount: 100},
			wantErr: ErrInvalidTransactionType,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.tx.Validate()
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestTransactionType_IsValid(t *testing.T) {
	cases := []struct {
		name    string
		trType  TransactionType
		isValid bool
	}{
		{"ok/valid_bet", TransactionTypeBet, true},
		{"ok/valid_win", TransactionTypeWin, true},
		{"err/invalid_type", "unknown", false},
		{"err/empty_type", "", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.trType.IsValid()
			if (err == nil) != tc.isValid {
				t.Errorf("IsValid() for %s was %v, want %v", tc.trType, (err == nil), tc.isValid)
			}
		})
	}
}
