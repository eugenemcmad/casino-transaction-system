package repository

import (
	"casino-transaction-system/internal/domain"
	basemetrics "casino-transaction-system/internal/observability/metrics"
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockMetricsSink struct {
	mu        sync.Mutex
	counters  map[string]int64
	durations map[string]int
}

func newMockMetricsSink() *mockMetricsSink {
	return &mockMetricsSink{
		counters:  make(map[string]int64),
		durations: make(map[string]int),
	}
}

func (m *mockMetricsSink) IncCounter(name string, labels basemetrics.Labels, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[basemetrics.Key(name, labels)] += value
}

func (m *mockMetricsSink) SetGauge(name string, labels basemetrics.Labels, value float64) {}

func (m *mockMetricsSink) ObserveDuration(name string, labels basemetrics.Labels, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := basemetrics.Key(name, labels)
	m.durations[key]++
}

func (m *mockMetricsSink) Flush() {}

func mustCloseSQLDB(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Logf("db.Close() returned error: %v", err)
	}
}

func TestNewPostgresRepo(t *testing.T) {
	cases := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{name: "err/returns_error_for_invalid_dsn", dsn: "://invalid-dsn", wantErr: true},
		{name: "err/returns_error_for_unreachable_db", dsn: "postgres://user:pass@127.0.0.1:1/testdb?sslmode=disable", wantErr: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			repo, err := NewPostgresRepo(tc.dsn)
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewPostgresRepo() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr && !errors.Is(err, ErrDBUnavailable) {
				t.Fatalf("NewPostgresRepo() expected ErrDBUnavailable, got %v", err)
			}
			if repo != nil {
				repo.Close()
			}
		})
	}
}

func TestPostgresRepo_SaveBulk(t *testing.T) {
	t.Run("ok/empty_slice", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		repo := &PostgresRepo{db: db}
		if err := repo.SaveBulk(context.Background(), nil); err != nil {
			t.Fatalf("SaveBulk(nil) error = %v", err)
		}
		if err := repo.SaveBulk(context.Background(), []domain.Transaction{}); err != nil {
			t.Fatalf("SaveBulk(empty) error = %v", err)
		}
		mock.ExpectClose()
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql expectations: %v", err)
		}
	})

	t.Run("ok/inserts_multiple_rows", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}

		repo := &PostgresRepo{db: db}
		ts := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
		txs := []domain.Transaction{
			{UserID: 1, Type: domain.TransactionTypeBet, Amount: 100, Timestamp: ts},
			{UserID: 2, Type: domain.TransactionTypeWin, Amount: 200},
		}

		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(
				int64(1), string(domain.TransactionTypeBet), int64(100), sqlmock.AnyArg(),
				int64(2), string(domain.TransactionTypeWin), int64(200), sql.NullTime{},
			).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectClose()

		if err := repo.SaveBulk(context.Background(), txs); err != nil {
			t.Fatalf("SaveBulk() error = %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql expectations: %v", err)
		}
	})

	t.Run("err/exec_failure", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}

		repo := &PostgresRepo{db: db}
		txs := []domain.Transaction{
			{UserID: 1, Type: domain.TransactionTypeBet, Amount: 50},
		}

		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(int64(1), string(domain.TransactionTypeBet), int64(50), sql.NullTime{}).
			WillReturnError(errors.New("bulk insert failed"))
		mock.ExpectClose()

		if err := repo.SaveBulk(context.Background(), txs); err == nil {
			t.Fatal("SaveBulk() expected error, got nil")
		}
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql expectations: %v", err)
		}
	})

	t.Run("err/uninitialized_repo", func(t *testing.T) {
		repo := &PostgresRepo{}
		err := repo.SaveBulk(context.Background(), []domain.Transaction{{UserID: 1, Type: domain.TransactionTypeBet, Amount: 1}})
		if !errors.Is(err, ErrRepoNotInitialized) {
			t.Fatalf("SaveBulk() error = %v, want %v", err, ErrRepoNotInitialized)
		}
	})
}

func TestPostgresRepo_SaveBulk_EmitsMetrics(t *testing.T) {
	t.Run("ok/emits_success_counter_and_duration", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}

		metrics := newMockMetricsSink()
		repo := &PostgresRepo{db: db, metrics: metrics}
		txs := []domain.Transaction{
			{UserID: 1, Type: domain.TransactionTypeBet, Amount: 10},
		}

		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(int64(1), string(domain.TransactionTypeBet), int64(10), sql.NullTime{}).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectClose()

		if err := repo.SaveBulk(context.Background(), txs); err != nil {
			t.Fatalf("SaveBulk() error = %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}

		successKey := basemetrics.Key(MetricPostgresQueriesTotal, basemetrics.Labels{
			"operation": "save_bulk",
			"result":    "success",
		})
		if metrics.counters[successKey] != 1 {
			t.Fatalf("success counter = %d, want 1", metrics.counters[successKey])
		}

		durationKey := basemetrics.Key(MetricPostgresQueryDurationMs, basemetrics.Labels{"operation": "save_bulk"})
		if metrics.durations[durationKey] != 1 {
			t.Fatalf("save_bulk duration observations = %d, want 1", metrics.durations[durationKey])
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql expectations: %v", err)
		}
	})

	t.Run("err/emits_error_counter_on_exec", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}

		metrics := newMockMetricsSink()
		repo := &PostgresRepo{db: db, metrics: metrics}
		txs := []domain.Transaction{
			{UserID: 1, Type: domain.TransactionTypeWin, Amount: 5},
		}

		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(int64(1), string(domain.TransactionTypeWin), int64(5), sql.NullTime{}).
			WillReturnError(errors.New("exec failed"))
		mock.ExpectClose()

		if err := repo.SaveBulk(context.Background(), txs); err == nil {
			t.Fatal("SaveBulk() expected error, got nil")
		}
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}

		errorKey := basemetrics.Key(MetricPostgresQueriesTotal, basemetrics.Labels{
			"operation": "save_bulk",
			"result":    "error",
			"reason":    "exec",
		})
		if metrics.counters[errorKey] != 1 {
			t.Fatalf("error counter = %d, want 1", metrics.counters[errorKey])
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("sql expectations: %v", err)
		}
	})
}

func TestPostgresRepo_Save(t *testing.T) {
	t.Run("ok/saves_with_timestamp", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		repo := &PostgresRepo{db: db}
		tx := domain.Transaction{
			UserID:    10,
			Type:      domain.TransactionTypeBet,
			Amount:    1550,
			Timestamp: time.Now().UTC(),
		}

		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(tx.UserID, string(tx.Type), tx.Amount, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		if err := repo.Save(context.Background(), tx); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet sql expectations: %v", err)
		}
	})

	t.Run("err/returns_db_error_on_insert", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		repo := &PostgresRepo{db: db}
		tx := domain.Transaction{
			UserID: 1,
			Type:   domain.TransactionTypeWin,
			Amount: 125,
		}

		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(tx.UserID, string(tx.Type), tx.Amount, sql.NullTime{}).
			WillReturnError(errors.New("insert failed"))

		if err := repo.Save(context.Background(), tx); err == nil {
			t.Fatal("Save() expected error, got nil")
		}
	})
}

func TestPostgresRepo_Save_EmitsMetrics(t *testing.T) {
	t.Run("ok/emits_success_counter_and_duration", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		metrics := newMockMetricsSink()
		repo := &PostgresRepo{db: db, metrics: metrics}
		tx := domain.Transaction{
			UserID: 1,
			Type:   domain.TransactionTypeBet,
			Amount: 100,
		}

		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(tx.UserID, string(tx.Type), tx.Amount, sql.NullTime{}).
			WillReturnResult(sqlmock.NewResult(1, 1))

		if err := repo.Save(context.Background(), tx); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		successKey := basemetrics.Key(MetricPostgresQueriesTotal, basemetrics.Labels{
			"operation": "save",
			"result":    "success",
		})
		if metrics.counters[successKey] != 1 {
			t.Fatalf("success counter = %d, want 1", metrics.counters[successKey])
		}

		durationKey := basemetrics.Key(MetricPostgresQueryDurationMs, basemetrics.Labels{"operation": "save"})
		if metrics.durations[durationKey] != 1 {
			t.Fatalf("save duration observations = %d, want 1", metrics.durations[durationKey])
		}
	})

	t.Run("err/emits_error_counter_and_duration", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		metrics := newMockMetricsSink()
		repo := &PostgresRepo{db: db, metrics: metrics}
		tx := domain.Transaction{
			UserID: 1,
			Type:   domain.TransactionTypeWin,
			Amount: 200,
		}

		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(tx.UserID, string(tx.Type), tx.Amount, sql.NullTime{}).
			WillReturnError(errors.New("insert failed"))

		if err := repo.Save(context.Background(), tx); err == nil {
			t.Fatal("Save() expected error, got nil")
		}

		errorKey := basemetrics.Key(MetricPostgresQueriesTotal, basemetrics.Labels{
			"operation": "save",
			"result":    "error",
			"reason":    "exec",
		})
		if metrics.counters[errorKey] != 1 {
			t.Fatalf("error counter = %d, want 1", metrics.counters[errorKey])
		}

		durationKey := basemetrics.Key(MetricPostgresQueryDurationMs, basemetrics.Labels{"operation": "save"})
		if metrics.durations[durationKey] != 1 {
			t.Fatalf("save duration observations = %d, want 1", metrics.durations[durationKey])
		}
	})
}

func TestPostgresRepo_Get(t *testing.T) {
	t.Run("ok/returns_rows_with_nullable_timestamp", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		repo := &PostgresRepo{db: db}
		txType := domain.TransactionTypeBet
		expectedQuery := QueryGetTransactionsBase +
			" AND user_id = $1 AND type = $2" +
			QueryOrderByTimestampDesc

		now := time.Now().UTC()
		rows := sqlmock.NewRows([]string{"user_id", "type", "amount", "timestamp", "created_at"}).
			AddRow(int64(7), "bet", int64(1370), now, now).
			AddRow(int64(7), "win", int64(2000), nil, now)

		mock.ExpectQuery(expectedQuery).
			WithArgs(int64(7), "bet").
			WillReturnRows(rows)

		got, err := repo.Get(context.Background(), 7, &txType)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("Get() len = %d, want 2", len(got))
		}
		if got[0].Timestamp.IsZero() {
			t.Fatal("expected first timestamp to be set")
		}
		if !got[1].Timestamp.IsZero() {
			t.Fatal("expected second timestamp to be zero for NULL DB value")
		}
	})

	t.Run("err/returns_query_error", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		repo := &PostgresRepo{db: db}
		expectedQuery := QueryGetTransactionsBase +
			" AND user_id = $1" +
			QueryOrderByTimestampDesc

		mock.ExpectQuery(expectedQuery).
			WithArgs(int64(99)).
			WillReturnError(errors.New("query failed"))

		_, err = repo.Get(context.Background(), 99, nil)
		if err == nil {
			t.Fatal("Get() expected error, got nil")
		}
	})

	t.Run("err/returns_scan_error", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		repo := &PostgresRepo{db: db}
		expectedQuery := QueryGetTransactionsBase + QueryOrderByTimestampDesc
		rows := sqlmock.NewRows([]string{"user_id", "type", "amount", "timestamp", "created_at"}).
			AddRow("bad-user-id", "bet", int64(1000), time.Now().UTC(), time.Now().UTC())

		mock.ExpectQuery(expectedQuery).WillReturnRows(rows)

		_, err = repo.Get(context.Background(), 0, nil)
		if err == nil {
			t.Fatal("Get() expected scan error, got nil")
		}
	})
}

func TestPostgresRepo_Get_EmitsMetrics(t *testing.T) {
	t.Run("ok/emits_success_counter_and_duration", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		metrics := newMockMetricsSink()
		repo := &PostgresRepo{db: db, metrics: metrics}

		expectedQuery := QueryGetTransactionsBase + QueryOrderByTimestampDesc
		rows := sqlmock.NewRows([]string{"user_id", "type", "amount", "timestamp", "created_at"}).
			AddRow(int64(7), "bet", int64(1370), nil, time.Now().UTC())

		mock.ExpectQuery(expectedQuery).WillReturnRows(rows)

		if _, err := repo.Get(context.Background(), 0, nil); err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		successKey := basemetrics.Key(MetricPostgresQueriesTotal, basemetrics.Labels{
			"operation": "get",
			"result":    "success",
		})
		if metrics.counters[successKey] != 1 {
			t.Fatalf("success counter = %d, want 1", metrics.counters[successKey])
		}

		durationKey := basemetrics.Key(MetricPostgresQueryDurationMs, basemetrics.Labels{"operation": "get"})
		if metrics.durations[durationKey] != 1 {
			t.Fatalf("get duration observations = %d, want 1", metrics.durations[durationKey])
		}
	})

	t.Run("err/emits_query_error_counter_and_duration", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		metrics := newMockMetricsSink()
		repo := &PostgresRepo{db: db, metrics: metrics}

		expectedQuery := QueryGetTransactionsBase + QueryOrderByTimestampDesc
		mock.ExpectQuery(expectedQuery).WillReturnError(errors.New("query failed"))

		if _, err := repo.Get(context.Background(), 0, nil); err == nil {
			t.Fatal("Get() expected error, got nil")
		}

		errorKey := basemetrics.Key(MetricPostgresQueriesTotal, basemetrics.Labels{
			"operation": "get",
			"result":    "error",
			"reason":    "query",
		})
		if metrics.counters[errorKey] != 1 {
			t.Fatalf("error counter = %d, want 1", metrics.counters[errorKey])
		}

		durationKey := basemetrics.Key(MetricPostgresQueryDurationMs, basemetrics.Labels{"operation": "get"})
		if metrics.durations[durationKey] != 1 {
			t.Fatalf("get duration observations = %d, want 1", metrics.durations[durationKey])
		}
	})

	t.Run("err/rows_close_error_propagates_as_rows_error", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		metrics := newMockMetricsSink()
		repo := &PostgresRepo{db: db, metrics: metrics}

		expectedQuery := QueryGetTransactionsBase + QueryOrderByTimestampDesc
		now := time.Now().UTC()
		rows := sqlmock.NewRows([]string{"user_id", "type", "amount", "timestamp", "created_at"}).
			AddRow(int64(7), "bet", int64(100), now, now).
			CloseError(errors.New("close rows failed"))

		mock.ExpectQuery(expectedQuery).WillReturnRows(rows)

		_, err = repo.Get(context.Background(), 0, nil)
		if err == nil {
			t.Fatal("Get() expected error, got nil")
		}

		rowsErrKey := basemetrics.Key(MetricPostgresQueriesTotal, basemetrics.Labels{
			"operation": "get",
			"result":    "error",
			"reason":    "rows",
		})
		if metrics.counters[rowsErrKey] != 1 {
			t.Fatalf("rows error counter = %d, want 1", metrics.counters[rowsErrKey])
		}
	})
}

func TestPostgresRepo_Close(t *testing.T) {
	t.Run("ok/closes_non_nil_db", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}

		repo := &PostgresRepo{db: db}
		mock.ExpectClose()
		repo.Close()

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet sql expectations: %v", err)
		}
	})

	t.Run("ok/handles_nil_db", func(t *testing.T) {
		repo := &PostgresRepo{}
		repo.Close()
	})

	t.Run("err/close_db_error_emits_counter", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer mustCloseSQLDB(t, db)

		metrics := newMockMetricsSink()
		repo := &PostgresRepo{db: db, metrics: metrics}

		mock.ExpectClose().WillReturnError(errors.New("close db failed"))
		repo.Close()

		closeDBKey := basemetrics.Key(MetricPostgresQueriesTotal, basemetrics.Labels{
			"operation": "close",
			"result":    "error",
			"reason":    "close_db",
		})
		if metrics.counters[closeDBKey] != 1 {
			t.Fatalf("close_db counter = %d, want 1", metrics.counters[closeDBKey])
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet sql expectations: %v", err)
		}
	})
}

func TestPostgresRepo_Save_ReturnsErrorForUninitializedRepo(t *testing.T) {
	repo := &PostgresRepo{}
	tx := domain.Transaction{UserID: 1, Type: domain.TransactionTypeBet, Amount: 10}

	err := repo.Save(context.Background(), tx)
	if !errors.Is(err, ErrRepoNotInitialized) {
		t.Fatalf("Save() error = %v, wantErr %v", err, ErrRepoNotInitialized)
	}
}

func TestPostgresRepo_Get_ReturnsErrorForUninitializedRepo(t *testing.T) {
	repo := &PostgresRepo{}

	_, err := repo.Get(context.Background(), 1, nil)
	if !errors.Is(err, ErrRepoNotInitialized) {
		t.Fatalf("Get() error = %v, wantErr %v", err, ErrRepoNotInitialized)
	}
}
