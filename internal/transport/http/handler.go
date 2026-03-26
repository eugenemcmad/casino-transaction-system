// Package http exposes REST handlers and JSON DTOs for the transaction API.
package http

import (
	"casino-transaction-system/internal/boundary"
	"casino-transaction-system/internal/domain"
	basemetrics "casino-transaction-system/internal/observability/metrics"
	"casino-transaction-system/internal/service"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// TransactionHandler serves HTTP endpoints backed by TransactionService.
type TransactionHandler struct {
	svc     service.TransactionService
	metrics basemetrics.Sink
}

// NewTransactionHandler creates a handler with the given service.
func NewTransactionHandler(svc service.TransactionService) *TransactionHandler {
	return NewTransactionHandlerWithMetrics(svc, basemetrics.NewLogSink())
}

// NewTransactionHandlerWithMetrics creates a handler with service and reusable metrics sink.
func NewTransactionHandlerWithMetrics(svc service.TransactionService, metricsSink basemetrics.Sink) *TransactionHandler {
	if metricsSink == nil {
		metricsSink = basemetrics.NewLogSink()
	}
	slog.Debug("Initializing TransactionHandler")
	return &TransactionHandler{svc: svc, metrics: metricsSink}
}

// GetTransactions handles GET /transactions with optional user_id and transaction_type query filters.
//
// @Summary Get transactions
// @Description Returns transactions with optional filters by user_id and transaction_type.
// @Tags transactions
// @Produce json
// @Param user_id query int false "Filter by user ID (>0)"
// @Param transaction_type query string false "Filter by type" Enums(bet,win)
// @Success 200 {array} TransactionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /transactions [get]
func (h *TransactionHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	defer h.observeDuration("get_transactions", startedAt)
	defer h.flushMetrics()

	slog.Debug("HTTP GetTransactions request received", "query", r.URL.RawQuery)
	userIDStr := r.URL.Query().Get("user_id")
	tTypeStr := r.URL.Query().Get("transaction_type")

	var userID int64
	var err error
	if userIDStr != "" {
		userID, err = strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			slog.Warn(MsgInvalidUserID, "userIDStr", userIDStr)
			h.incRequestCounter(http.StatusBadRequest, "invalid_user_id")
			http.Error(w, MsgInvalidUserID, http.StatusBadRequest)
			return
		}
		if userID <= 0 {
			slog.Warn(MsgInvalidUserID, "userID", userID)
			h.incRequestCounter(http.StatusBadRequest, "non_positive_user_id")
			http.Error(w, MsgInvalidUserID, http.StatusBadRequest)
			return
		}
	}

	var tType *domain.TransactionType
	if tTypeStr != "" {
		typ := domain.TransactionType(tTypeStr)
		if err := typ.IsValid(); err != nil {
			slog.Warn(MsgInvalidTransactionTypeInReq, "tTypeStr", tTypeStr)
			h.incRequestCounter(http.StatusBadRequest, "invalid_transaction_type")
			http.Error(w, MsgInvalidTransactionTypeInReq, http.StatusBadRequest)
			return
		}
		tType = &typ
	}

	transactions, err := h.svc.GetTransactions(r.Context(), userID, tType)
	if err != nil {
		h.incRequestCounter(boundary.Classify(err).HTTPStatus, "service_error")
		writeServiceError(w, err)
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
		h.incRequestCounter(http.StatusInternalServerError, "encode_error")
		return
	}
	h.incRequestCounter(http.StatusOK, "ok")
}

func (h *TransactionHandler) incRequestCounter(statusCode int, reason string) {
	if h == nil || h.metrics == nil {
		return
	}
	labels := basemetrics.Labels{
		"endpoint": "/transactions",
		"method":   http.MethodGet,
		"status":   strconv.Itoa(statusCode),
	}
	if reason != "" {
		labels["reason"] = reason
	}
	h.metrics.IncCounter(MetricAPIRequestsTotal, labels, 1)
}

func (h *TransactionHandler) observeDuration(endpoint string, startedAt time.Time) {
	if h == nil || h.metrics == nil {
		return
	}
	h.metrics.ObserveDuration(MetricAPIRequestDurationMs, basemetrics.Labels{
		"endpoint": endpoint,
		"method":   http.MethodGet,
	}, time.Since(startedAt))
}

func (h *TransactionHandler) flushMetrics() {
	if h == nil || h.metrics == nil {
		return
	}
	h.metrics.Flush()
}
