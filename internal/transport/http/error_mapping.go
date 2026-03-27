package http

import (
	"casino-transaction-system/internal/boundary"
	"log/slog"
	"net/http"
)

func writeServiceError(w http.ResponseWriter, err error) {
	meta := boundary.Classify(err)
	if meta.LogLevel == slog.LevelWarn {
		slog.Warn(MsgFailedToGetTransactions, "error", err, "error_code", meta.Code)
	} else {
		slog.Error(MsgFailedToGetTransactions, "error", err, "error_code", meta.Code)
	}
	http.Error(w, MsgFailedToGetTransactions, meta.HTTPStatus)
}
