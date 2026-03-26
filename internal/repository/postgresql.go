// Package repository implements domain.TransactionRepository against PostgreSQL.
package repository

import (
	"casino-transaction-system/internal/domain"
	basemetrics "casino-transaction-system/internal/observability/metrics"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PostgresRepo persists transactions using database/sql and the pq driver.
type PostgresRepo struct {
	db      *sql.DB
	metrics basemetrics.Sink
}

// PoolConfig configures sql.DB connection pool limits and connection lifetime.
type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DefaultPoolConfig is used when NewPostgresRepo is called without explicit pool settings.
var DefaultPoolConfig = PoolConfig{
	MaxOpenConns:    25,
	MaxIdleConns:    5,
	ConnMaxLifetime: 5 * time.Minute,
}

// NewPostgresRepo initializes DB and validates connectivity.
// It fails fast if database initialization is not possible.
func NewPostgresRepo(url string) (*PostgresRepo, error) {
	return NewPostgresRepoWithPoolAndMetrics(url, DefaultPoolConfig, basemetrics.NewLogSink())
}

// NewPostgresRepoWithPoolAndMetrics opens a pool, applies poolCfg, pings DB, and wires metrics sink.
func NewPostgresRepoWithPoolAndMetrics(url string, poolCfg PoolConfig, metricsSink basemetrics.Sink) (*PostgresRepo, error) {
	if metricsSink == nil {
		metricsSink = basemetrics.NewLogSink()
	}
	slog.Debug(MsgInitializingPostgres, "url", url)
	metricsSink.IncCounter(MetricPostgresConnectionAttempts, basemetrics.Labels{"result": "started"}, 1)

	db, err := sql.Open(DriverPostgres, url)
	if err != nil {
		slog.Error(MsgErrorOpeningDB, "error", err)
		metricsSink.IncCounter(MetricPostgresConnectionAttempts, basemetrics.Labels{"result": "error", "reason": "open"}, 1)
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
		metricsSink.IncCounter(MetricPostgresConnectionAttempts, basemetrics.Labels{"result": "error", "reason": "ping"}, 1)
		if closeErr := db.Close(); closeErr != nil {
			slog.Warn(MsgErrorClosingDB, "error", closeErr)
			metricsSink.IncCounter(MetricPostgresConnectionAttempts, basemetrics.Labels{"result": "error", "reason": "close_db"}, 1)
		}
		return nil, fmt.Errorf("%w: ping postgres: %v", ErrDBUnavailable, err)
	}
	metricsSink.IncCounter(MetricPostgresConnectionAttempts, basemetrics.Labels{"result": "success"}, 1)
	slog.Debug(MsgDBConnectionSuccess)

	return &PostgresRepo{db: db, metrics: metricsSink}, nil
}

// Save inserts a transaction; duplicates matching the unique key are ignored (ON CONFLICT DO NOTHING).
func (r *PostgresRepo) Save(ctx context.Context, t domain.Transaction) error {
	startedAt := time.Now()
	defer r.observeDuration("save", startedAt)

	if r == nil || r.db == nil {
		r.incQueryCounter("save", "error", "repo_not_initialized")
		return ErrRepoNotInitialized
	}
	slog.Debug(MsgSavingToDB, "userID", t.UserID, "type", t.Type, "amount", t.Amount)

	var ts sql.NullTime
	if !t.Timestamp.IsZero() {
		ts.Time = t.Timestamp
		ts.Valid = true
	}

	query := `
		INSERT INTO transactions (user_id, type, amount, timestamp)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, type, amount, timestamp) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, t.UserID, string(t.Type), t.Amount, ts)
	if err != nil {
		slog.Error(MsgFailedToInsert, "error", err, "transaction", t)
		r.incQueryCounter("save", "error", "exec")
		return err
	}
	r.incQueryCounter("save", "success", "")
	slog.Debug(MsgTransactionSaved, "userID", t.UserID)
	return nil
}

// SaveBulk inserts multiple transactions efficiently in a single query.
// Duplicates matching the unique key are ignored (ON CONFLICT DO NOTHING).
// If the entire bulk insert fails, it returns an error.
func (r *PostgresRepo) SaveBulk(ctx context.Context, txs []domain.Transaction) error {
	startedAt := time.Now()
	defer r.observeDuration("save_bulk", startedAt)

	if r == nil || r.db == nil {
		r.incQueryCounter("save_bulk", "error", "repo_not_initialized")
		return ErrRepoNotInitialized
	}

	if len(txs) == 0 {
		return nil
	}

	slog.Debug("Saving transactions in bulk", "count", len(txs))

	var valueStrings []string
	var valueArgs []any
	for i, t := range txs {
		// ($1, $2, $3, $4), ($5, $6, $7, $8)...
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4))
		var ts sql.NullTime
		if !t.Timestamp.IsZero() {
			ts.Time = t.Timestamp
			ts.Valid = true
		}
		valueArgs = append(valueArgs, t.UserID, string(t.Type), t.Amount, ts)
	}

	query := fmt.Sprintf(`
		INSERT INTO transactions (user_id, type, amount, timestamp)
		VALUES %s
		ON CONFLICT (user_id, type, amount, timestamp) DO NOTHING
	`, strings.Join(valueStrings, ","))

	_, err := r.db.ExecContext(ctx, query, valueArgs...)
	if err != nil {
		slog.Error("Failed to bulk insert transactions", "error", err, "count", len(txs))
		r.incQueryCounter("save_bulk", "error", "exec")
		return err
	}

	r.incQueryCounter("save_bulk", "success", "")
	slog.Debug("Transactions bulk saved successfully", "count", len(txs))
	return nil
}

// Get fetches transactions based on optional filters.
func (r *PostgresRepo) Get(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	startedAt := time.Now()
	defer r.observeDuration("get", startedAt)

	if r == nil || r.db == nil {
		r.incQueryCounter("get", "error", "repo_not_initialized")
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
		r.incQueryCounter("get", "error", "query")
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			slog.Warn(MsgFailedToCloseRows, "error", closeErr)
			r.incQueryCounter("get", "error", "close_rows")
		}
	}()

	var transactions []domain.Transaction
	for rows.Next() {
		var t domain.Transaction
		var tTypeStr string
		var amount int64
		var ts sql.NullTime

		if err := rows.Scan(&t.UserID, &tTypeStr, &amount, &ts, &t.CreatedAt); err != nil {
			slog.Error(MsgFailedToScanRow, "error", err)
			r.incQueryCounter("get", "error", "scan")
			return nil, err
		}

		t.Type = domain.TransactionType(tTypeStr)
		t.Amount = amount
		if ts.Valid {
			t.Timestamp = ts.Time
		}

		transactions = append(transactions, t)
	}
	if err := rows.Err(); err != nil {
		slog.Error(MsgRowsError, "error", err)
		r.incQueryCounter("get", "error", "rows")
		return nil, err
	}

	r.incQueryCounter("get", "success", "")
	slog.Debug(MsgFetchedCount, "userID", userID, "count", len(transactions))
	return transactions, nil
}

// Close closes the underlying database handle.
func (r *PostgresRepo) Close() {
	if r == nil || r.db == nil {
		return
	}

	slog.Debug(MsgClosingPostgres)
	if err := r.db.Close(); err != nil {
		slog.Warn(MsgErrorClosingDB, "error", err)
		r.incQueryCounter("close", "error", "close_db")
	}
}

func (r *PostgresRepo) incQueryCounter(operation, result, reason string) {
	if r == nil || r.metrics == nil {
		return
	}
	labels := basemetrics.Labels{
		"operation": operation,
		"result":    result,
	}
	if reason != "" {
		labels["reason"] = reason
	}
	r.metrics.IncCounter(MetricPostgresQueriesTotal, labels, 1)
}

func (r *PostgresRepo) observeDuration(operation string, startedAt time.Time) {
	if r == nil || r.metrics == nil {
		return
	}
	r.metrics.ObserveDuration(MetricPostgresQueryDurationMs, basemetrics.Labels{"operation": operation}, time.Since(startedAt))
}
