package domain

import "errors"

type TransactionType string

const (
	TransactionTypeBet TransactionType = "bet"
	TransactionTypeWin TransactionType = "win"
)

var ErrInvalidTransactionType = errors.New("invalid transaction type")

func (t TransactionType) IsValid() error {
	switch t {
	case TransactionTypeBet, TransactionTypeWin:
		return nil
	}
	return ErrInvalidTransactionType
}
