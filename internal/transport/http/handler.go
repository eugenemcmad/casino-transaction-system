package http

import (
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/internal/service"
	"encoding/json"
	"io"
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

// CreateTransaction exists for TESTING purposes only (as per tech specs).
func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	slog.Debug("HTTP CreateTransaction request received")
	
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read request body", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var req CreateTransactionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		slog.Warn("HTTP Body Unmarshal failed (REJECTED)", 
			"error", err, 
			"raw_payload", string(body),
		)
		http.Error(w, MsgInvalidRequestBody, http.StatusBadRequest)
		return
	}

	t := req.ToDomain()
	
	if err := t.Validate(); err != nil {
		slog.Warn("HTTP transaction validation failed (REJECTED)", 
			"error", err, 
			"reason", err.Error(),
			"raw_payload", string(body),
		)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Timestamp == "" {
		slog.Warn(MsgMissingZeroTimestamp, "userID", req.UserID)
	}

	if err := h.svc.RegisterTransaction(r.Context(), t); err != nil {
		slog.Error(MsgFailedToRegisterTransaction, "error", err)
		http.Error(w, MsgFailedToRegisterTransaction, http.StatusInternalServerError)
		return
	}

	slog.Info(MsgTransactionProcessed, "userID", t.UserID, "type", t.Type, "amount", t.Amount)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": StatusRegistered})
}

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
