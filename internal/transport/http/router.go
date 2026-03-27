package http

import (
	"log/slog"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

func NewRouter(handler *TransactionHandler) *http.ServeMux {
	mux := http.NewServeMux()

	// GET: Transaction history retrieval
	mux.HandleFunc("GET /transactions", handler.GetTransactions)

	// GET: Health check endpoint
	mux.HandleFunc("GET /health", healthHandler)

	// Swagger UI
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	return mux
}

// healthHandler godoc
// @Summary Health check
// @Description Returns service health status.
// @Tags health
// @Success 200 {string} string "OK"
// @Router /health [get]
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		slog.Error("Failed to write response", "error", err)
	}
}
