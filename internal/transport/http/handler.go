package http

import (
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/internal/service"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

type TransactionHandler struct {
	svc service.TransactionService
}

func NewTransactionHandler(svc service.TransactionService) *TransactionHandler {
	slog.Debug("Initializing TransactionHandler")
	return &TransactionHandler{svc: svc}
}

// GetTransactions retrieves the transaction history.
// Supports filtering by user_id and transaction_type.
func (h *TransactionHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	slog.Debug("HTTP GetTransactions request received", "query", r.URL.RawQuery)
	userIDStr := r.URL.Query().Get("user_id")
	tTypeStr := r.URL.Query().Get("transaction_type")

	var userID int64
	var err error
	if userIDStr != "" {
		userID, err = strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			slog.Warn(MsgInvalidUserID, "userIDStr", userIDStr)
			http.Error(w, MsgInvalidUserID, http.StatusBadRequest)
			return
		}
	}

	var tType *domain.TransactionType
	if tTypeStr != "" {
		typ := domain.TransactionType(tTypeStr)
		if err := typ.IsValid(); err != nil {
			slog.Warn(MsgInvalidTransactionTypeInReq, "tTypeStr", tTypeStr)
			http.Error(w, MsgInvalidTransactionTypeInReq, http.StatusBadRequest)
			return
		}
		tType = &typ
	}

	transactions, err := h.svc.GetTransactions(r.Context(), userID, tType)
	if err != nil {
		slog.Error(MsgFailedToGetTransactions, "error", err, "userID", userID)
		http.Error(w, MsgFailedToGetTransactions, http.StatusInternalServerError)
		return
	}

	resp := make([]TransactionResponse, 0, len(transactions))
	for _, t := range transactions {
		resp = append(resp, NewTransactionResponse(t))
	}

	slog.Debug("Sending transactions response", "count", len(resp))
	w.Header().Set(HeaderContentType, MimeApplicationJSON)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}
