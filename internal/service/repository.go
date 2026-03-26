package service

import (
	"casino-transaction-system/internal/domain"
	"context"
	"log/slog"
)

// transactionService implementation of TransactionService interface.
type transactionService struct {
	repo TransactionRepository
}

func NewTransactionService(repo TransactionRepository) TransactionService {
	slog.Debug("Initializing transactionService")
	return &transactionService{repo: repo}
}

func (s *transactionService) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	slog.Debug("Registering transaction", "userID", t.UserID, "type", t.Type, "amount", t.Amount)
	return s.repo.Save(ctx, t)
}

func (s *transactionService) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	slog.Debug("Getting transactions history", "userID", userID, "type", tType)
	return s.repo.GetByUserID(ctx, userID, tType)
}
