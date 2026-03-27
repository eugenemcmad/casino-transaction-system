package http

import (
	"log/slog"
	"net/http"
)

func NewRouter(handler *TransactionHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// GET: Transaction history retrieval
	mux.HandleFunc("GET /transactions", handler.GetTransactions)

	// GET: Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			slog.Error("Failed to write response", "error", err)
		}
	})

	return mux
}
