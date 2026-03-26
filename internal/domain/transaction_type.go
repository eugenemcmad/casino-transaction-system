package domain

type TransactionType string

const (
	TransactionTypeBet TransactionType = "bet"
	TransactionTypeWin TransactionType = "win"
)

func (t TransactionType) IsValid() error {
	switch t {
	case TransactionTypeBet, TransactionTypeWin:
		return nil
	}
	return ErrInvalidTransactionType
}
