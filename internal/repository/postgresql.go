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

type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

var DefaultPoolConfig = PoolConfig{
	MaxOpenConns:    25,
	MaxIdleConns:    5,
	ConnMaxLifetime: 5 * time.Minute,
}

// NewPostgresRepo initializes DB and validates connectivity.
// It fails fast if database initialization is not possible.
func NewPostgresRepo(url string) (*PostgresRepo, error) {
	return NewPostgresRepoWithPool(url, DefaultPoolConfig)
}

func NewPostgresRepoWithPool(url string, poolCfg PoolConfig) (*PostgresRepo, error) {
	slog.Debug(MsgInitializingPostgres, "url", url)
	db, err := sql.Open(DriverPostgres, url)
	if err != nil {
		slog.Error(MsgErrorOpeningDB, "error", err)
		return nil, fmt.Errorf("%w: open postgres connection: %v", ErrDBUnavailable, err)
	}

	if poolCfg.MaxOpenConns <= 0 {
		poolCfg.MaxOpenConns = DefaultPoolConfig.MaxOpenConns
	}
	if poolCfg.MaxIdleConns <= 0 {
		poolCfg.MaxIdleConns = DefaultPoolConfig.MaxIdleConns
	}
	if poolCfg.ConnMaxLifetime <= 0 {
		poolCfg.ConnMaxLifetime = DefaultPoolConfig.ConnMaxLifetime
	}

	db.SetMaxOpenConns(poolCfg.MaxOpenConns)
	db.SetMaxIdleConns(poolCfg.MaxIdleConns)
	db.SetConnMaxLifetime(poolCfg.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		slog.Error(MsgDBPingFailed, "error", err)
		_ = db.Close()
		return nil, fmt.Errorf("%w: ping postgres: %v", ErrDBUnavailable, err)
	}
	slog.Debug(MsgDBConnectionSuccess)

	return &PostgresRepo{db: db}, nil
}

func (r *PostgresRepo) Save(ctx context.Context, t domain.Transaction) error {
	if r == nil || r.db == nil {
		return ErrRepoNotInitialized
	}
	slog.Debug(MsgSavingToDB, "userID", t.UserID, "type", t.Type, "amount", t.Amount)

	var ts sql.NullTime
	if !t.Timestamp.IsZero() {
		ts.Time = t.Timestamp
		ts.Valid = true
	}

	// Idempotency: using business-key (user_id, type, amount, timestamp)
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

// Get fetches transactions based on optional filters.
func (r *PostgresRepo) Get(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepoNotInitialized
	}
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
