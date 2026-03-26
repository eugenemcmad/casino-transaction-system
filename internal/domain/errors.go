package domain

import "errors"

var (
	ErrInvalidTransactionType = errors.New("invalid transaction type: use 'bet' or 'win'")
	ErrInvalidUserID          = errors.New("user_id must be positive")
	ErrInvalidAmount          = errors.New("amount must be positive")
)
