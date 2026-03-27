package service

import (
	"casino-transaction-system/internal/domain"
	"context"
	"log/slog"
)

// transactionService is the default TransactionService implementation.
type transactionService struct {
	repo domain.TransactionRepository
}

// NewTransactionService wires the domain repository into the use case implementation.
func NewTransactionService(repo domain.TransactionRepository) TransactionService {
	slog.Debug("Initializing transactionService")
	return &transactionService{repo: repo}
}

func (s *transactionService) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	slog.Debug("Registering transaction", "userID", t.UserID, "type", t.Type, "amount", t.Amount)
	return s.repo.Save(ctx, t)
}

func (s *transactionService) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	slog.Debug("Getting transactions history", "userID", userID, "type", tType)
	return s.repo.Get(ctx, userID, tType)
}
