package repository

import (
	"casino-transaction-system/internal/domain"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

type PostgresRepo struct {
	db *sql.DB
}

func NewPostgresRepo(url string) *PostgresRepo {
	slog.Debug(MsgInitializingPostgres, "url", url)
	db, err := sql.Open(DriverPostgres, url)
	if err != nil {
		slog.Error(MsgErrorOpeningDB, "error", err)
		return nil
	}
	// Проверим подключение
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

	_, err := r.db.ExecContext(ctx, QueryInsertTransaction, t.UserID, string(t.Type), t.Amount, ts)
	if err != nil {
		slog.Error(MsgFailedToInsert, "error", err, "transaction", t)
		return err
	}
	slog.Debug(MsgTransactionSaved, "userID", t.UserID)
	return nil
}

// GetByUserID now supports optional userID filtering (if userID <= 0, returns all)
func (r *PostgresRepo) GetByUserID(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	slog.Debug(MsgFetchingFromDB, "userID", userID, "type", tType)
	
	query := QueryGetTransactionsBase
	var args []any
	argCount := 1

	if userID > 0 {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, userID)
		argCount++
	}

	if tType != nil {
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, string(*tType))
	}

	query += QueryOrderByTimestampDesc

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		slog.Error(MsgFailedToQuery, "error", err, "userID", userID)
		return nil, err
	}
	defer rows.Close()

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
		} else {
			t.Timestamp = time.Time{} 
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
		r.db.Close()
	}
}
