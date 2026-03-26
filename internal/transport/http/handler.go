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

func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	slog.Debug("HTTP CreateTransaction request received")
	
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn(MsgInvalidRequestBody, "error", err)
		http.Error(w, MsgInvalidRequestBody, http.StatusBadRequest)
		return
	}

	if err := req.TransactionType.IsValid(); err != nil {
		slog.Warn(MsgInvalidTransactionType, "type", req.TransactionType)
		http.Error(w, MsgInvalidTransactionType, http.StatusBadRequest)
		return
	}

	if req.UserID <= 0 || req.Amount <= 0 {
		slog.Warn(MsgUserIDAmountMustBePositive, "userID", req.UserID, "amount", req.Amount)
		http.Error(w, MsgUserIDAmountMustBePositive, http.StatusBadRequest)
		return
	}

	if req.Timestamp == "" {
		slog.Warn(MsgMissingZeroTimestamp, "userID", req.UserID)
	}

	if err := h.svc.RegisterTransaction(r.Context(), req.ToDomain()); err != nil {
		slog.Error(MsgFailedToRegisterTransaction, "error", err)
		http.Error(w, MsgFailedToRegisterTransaction, http.StatusInternalServerError)
		return
	}

	slog.Info(MsgTransactionProcessed, "userID", req.UserID, "type", req.TransactionType, "amount", req.Amount)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": StatusRegistered})
}

func (h *TransactionHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	slog.Debug("HTTP GetTransactions request received", "query", r.URL.RawQuery)
	userIDStr := r.URL.Query().Get("user_id")
	tTypeStr := r.URL.Query().Get("transaction_type")

	if userIDStr == "" {
		slog.Warn(MsgUserIDRequired)
		http.Error(w, MsgUserIDRequired, http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		slog.Warn(MsgInvalidUserID, "userIDStr", userIDStr)
		http.Error(w, MsgInvalidUserID, http.StatusBadRequest)
		return
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
