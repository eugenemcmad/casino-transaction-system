package domain

// TransactionType classifies a casino transaction as a bet or a win.
type TransactionType string

const (
	TransactionTypeBet TransactionType = "bet"
	TransactionTypeWin TransactionType = "win"
)

// IsValid returns nil when the type is one of the supported constants.
func (t TransactionType) IsValid() error {
	switch t {
	case TransactionTypeBet, TransactionTypeWin:
		return nil
	}
	return ErrInvalidTransactionType
}
