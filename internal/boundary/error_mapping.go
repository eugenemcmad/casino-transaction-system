// Package boundary maps infrastructure and parsing errors to HTTP-oriented metadata and stable codes.
package boundary

import (
	"casino-transaction-system/internal/repository"
	"casino-transaction-system/pkg/money"
	"errors"
	"log/slog"
	"net/http"
)

// ErrorMeta describes how an error should be surfaced to HTTP and logs (code, status, retry hints).
type ErrorMeta struct {
	Code       string
	HTTPStatus int
	LogLevel   slog.Level
	Reject     bool
	Retryable  bool
}

// Classify maps known errors to stable codes; unknown errors become internal_error.
func Classify(err error) ErrorMeta {
	switch {
	case errors.Is(err, repository.ErrRepoNotInitialized):
		return ErrorMeta{
			Code:       "repo_not_initialized",
			HTTPStatus: http.StatusServiceUnavailable,
			LogLevel:   slog.LevelError,
			Reject:     false,
			Retryable:  true,
		}
	case errors.Is(err, repository.ErrDBUnavailable):
		return ErrorMeta{
			Code:       "db_unavailable",
			HTTPStatus: http.StatusServiceUnavailable,
			LogLevel:   slog.LevelError,
			Reject:     false,
			Retryable:  true,
		}
	case errors.Is(err, money.ErrEmptyAmount),
		errors.Is(err, money.ErrInvalidAmount),
		errors.Is(err, money.ErrTooManyDecimals),
		errors.Is(err, money.ErrInvalidDecimalPart),
		errors.Is(err, money.ErrAmountOverflow):
		return ErrorMeta{
			Code:       "invalid_amount_format",
			HTTPStatus: http.StatusBadRequest,
			LogLevel:   slog.LevelWarn,
			Reject:     true,
			Retryable:  false,
		}
	default:
		return ErrorMeta{
			Code:       "internal_error",
			HTTPStatus: http.StatusInternalServerError,
			LogLevel:   slog.LevelError,
			Reject:     false,
			Retryable:  false,
		}
	}
}
