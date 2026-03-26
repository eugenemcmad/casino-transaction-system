package repository

import (
	"casino-transaction-system/internal/domain"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type PostgresRepo struct {
	db *sql.DB
}

// NewPostgresRepo initializes DB and sets up connection pool (Tech Lead improvement)
func NewPostgresRepo(url string) *PostgresRepo {
	slog.Debug(MsgInitializingPostgres, "url", url)
	db, err := sql.Open(DriverPostgres, url)
	if err != nil {
		slog.Error(MsgErrorOpeningDB, "error", err)
		return nil
	}

	// Pool configuration: production-ready settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		slog.Warn(MsgDBPingFailed, "error", err)
	} else {
		slog.Debug(MsgDBConnectionSuccess)
	}
	return &PostgresRepo{db: db}
}

func (r *PostgresRepo) Save(ctx context.Context, t domain.Transaction) error {
	slog.Debug(MsgSavingToDB, "userID", t.UserID, "type", t.Type, "amount", t.Amount)

	var ts sql.NullTime
	if !t.Timestamp.IsZero() {
		ts.Time = t.Timestamp
		ts.Valid = true
	}

	// Idempotency: using business-key (user_id, type, amount, timestamp)
	// ON CONFLICT DO NOTHING handles duplicates automatically (e.g. from Kafka retries)
	query := `
		INSERT INTO transactions (user_id, type, amount, timestamp)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, type, amount, timestamp) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, t.UserID, string(t.Type), t.Amount, ts)
	if err != nil {
		slog.Error(MsgFailedToInsert, "error", err, "transaction", t)
		return err
	}
	slog.Debug(MsgTransactionSaved, "userID", t.UserID)
	return nil
}

// GetByUserID fetches transactions based on optional filters.
func (r *PostgresRepo) GetByUserID(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	slog.Debug(MsgFetchingFromDB, "userID", userID, "type", tType)

	var conditions []string
	var args []any

	if userID > 0 {
		args = append(args, userID)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}

	if tType != nil {
		args = append(args, string(*tType))
		conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)))
	}

	query := QueryGetTransactionsBase
	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}
	query += QueryOrderByTimestampDesc

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		slog.Error(MsgFailedToQuery, "error", err, "userID", userID)
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			slog.Error(MsgFailedToCloseRows, "error", err)
		}
	}(rows)

	var transactions []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		var tTypeStr string
		var ts sql.NullTime

		if err := rows.Scan(&t.UserID, &tTypeStr, &t.Amount, &ts, &t.CreatedAt); err != nil {
			slog.Error(MsgFailedToScanRow, "error", err)
			return nil, err
		}

		t.Type = domain.TransactionType(tTypeStr)
		if ts.Valid {
			t.Timestamp = ts.Time
		}

		transactions = append(transactions, t)
	}
	if err := rows.Err(); err != nil {
		slog.Error(MsgRowsError, "error", err)
		return nil, err
	}

	slog.Debug(MsgFetchedCount, "userID", userID, "count", len(transactions))
	return transactions, nil
}

func (r *PostgresRepo) Close() {
	if r.db != nil {
		slog.Debug(MsgClosingPostgres)
		err := r.db.Close()
		if err != nil {
			slog.Error(MsgErrorClosingDB, "error", err)
		}
	}
}
