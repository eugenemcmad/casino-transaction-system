package repository

import (
	"casino-transaction-system/internal/domain"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

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
			if repo != nil {
				repo.Close()
			}
		})
	}
}

func TestPostgresRepo_Save(t *testing.T) {
	t.Run("ok/saves_with_timestamp", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := &PostgresRepo{db: db}
		tx := domain.Transaction{
			UserID:    10,
			Type:      domain.TransactionTypeBet,
			Amount:    15.5,
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
		defer db.Close()

		repo := &PostgresRepo{db: db}
		tx := domain.Transaction{
			UserID: 1,
			Type:   domain.TransactionTypeWin,
			Amount: 1.25,
		}

		mock.ExpectExec("INSERT INTO transactions").
			WithArgs(tx.UserID, string(tx.Type), tx.Amount, sql.NullTime{}).
			WillReturnError(errors.New("insert failed"))

		if err := repo.Save(context.Background(), tx); err == nil {
			t.Fatal("Save() expected error, got nil")
		}
	})
}

func TestPostgresRepo_Get(t *testing.T) {
	t.Run("ok/returns_rows_with_nullable_timestamp", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatalf("sqlmock.New() error = %v", err)
		}
		defer db.Close()

		repo := &PostgresRepo{db: db}
		txType := domain.TransactionTypeBet
		expectedQuery := QueryGetTransactionsBase +
			" AND user_id = $1 AND type = $2" +
			QueryOrderByTimestampDesc

		now := time.Now().UTC()
		rows := sqlmock.NewRows([]string{"user_id", "type", "amount", "timestamp", "created_at"}).
			AddRow(int64(7), "bet", 13.7, now, now).
			AddRow(int64(7), "win", 20.0, nil, now)

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
		defer db.Close()

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
		defer db.Close()

		repo := &PostgresRepo{db: db}
		expectedQuery := QueryGetTransactionsBase + QueryOrderByTimestampDesc
		rows := sqlmock.NewRows([]string{"user_id", "type", "amount", "timestamp", "created_at"}).
			AddRow("bad-user-id", "bet", 10.0, time.Now().UTC(), time.Now().UTC())

		mock.ExpectQuery(expectedQuery).WillReturnRows(rows)

		_, err = repo.Get(context.Background(), 0, nil)
		if err == nil {
			t.Fatal("Get() expected scan error, got nil")
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
}

func TestPostgresRepo_Save_ReturnsErrorForUninitializedRepo(t *testing.T) {
	repo := &PostgresRepo{}
	tx := domain.Transaction{UserID: 1, Type: domain.TransactionTypeBet, Amount: 10}

	err := repo.Save(context.Background(), tx)
	if err == nil {
		t.Fatal("Save() expected error for uninitialized repository, got nil")
	}
}

func TestPostgresRepo_Get_ReturnsErrorForUninitializedRepo(t *testing.T) {
	repo := &PostgresRepo{}

	_, err := repo.Get(context.Background(), 1, nil)
	if err == nil {
		t.Fatal("Get() expected error for uninitialized repository, got nil")
	}
}
