package http

import (
	"log/slog"
	"net/http"
)

func NewRouter(handler *TransactionHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// GET: История транзакций
	mux.HandleFunc("GET /transactions", handler.GetTransactions)

	// POST: TODO: RM: Создание транзакции (опционально, для удобства тестирования)
	mux.HandleFunc("POST /transactions", handler.CreateTransaction)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			slog.Error("Failed to write response", "error", err)
		}
	})

	return mux
}
