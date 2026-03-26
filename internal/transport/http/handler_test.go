package http

import (
	"casino-transaction-system/internal/domain"
	basemetrics "casino-transaction-system/internal/observability/metrics"
	"casino-transaction-system/internal/repository"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"
)

type mockService struct {
	getTransactionsFunc func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error)
}

func (m *mockService) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	return nil
}

func (m *mockService) RegisterTransactions(ctx context.Context, txs []domain.Transaction) error {
	return nil
}
func (m *mockService) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	if m.getTransactionsFunc != nil {
		return m.getTransactionsFunc(ctx, userID, tType)
	}
	return nil, nil
}

type mockHTTPMetricsSink struct {
	mu        sync.Mutex
	counters  map[string]int64
	durations map[string]int
	flushes   int
}

func newMockHTTPMetricsSink() *mockHTTPMetricsSink {
	return &mockHTTPMetricsSink{
		counters:  make(map[string]int64),
		durations: make(map[string]int),
	}
}

func (m *mockHTTPMetricsSink) IncCounter(name string, labels basemetrics.Labels, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[basemetrics.Key(name, labels)] += value
}

func (m *mockHTTPMetricsSink) SetGauge(name string, labels basemetrics.Labels, value float64) {}

func (m *mockHTTPMetricsSink) ObserveDuration(name string, labels basemetrics.Labels, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.durations[basemetrics.Key(name, labels)]++
}

func (m *mockHTTPMetricsSink) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flushes++
}

func TestTransactionHandler_GetTransactions_ReturnsExpectedStatusCodes(t *testing.T) {
	cases := []struct {
		name       string
		url        string
		setupMock  func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error)
		wantStatus int
	}{
		{
			name: "ok/returns_transactions_for_all_params",
			url:  "/transactions?user_id=1&transaction_type=bet",
			setupMock: func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
				return func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
					return []domain.Transaction{{UserID: 1, Type: "bet", Amount: 10}}, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "err/invalid_user_id",
			url:        "/transactions?user_id=abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "err/non_positive_user_id_zero",
			url:        "/transactions?user_id=0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "err/non_positive_user_id_negative",
			url:        "/transactions?user_id=-1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "err/invalid_transaction_type",
			url:        "/transactions?transaction_type=invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "err/service_failure",
			url:  "/transactions",
			setupMock: func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
				return func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
					return nil, errors.New("service error")
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "err/repo_not_initialized_maps_to_503",
			url:  "/transactions",
			setupMock: func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
				return func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
					return nil, repository.ErrRepoNotInitialized
				}
			},
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name: "err/db_unavailable_maps_to_503",
			url:  "/transactions",
			setupMock: func() func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
				return func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
					return nil, repository.ErrDBUnavailable
				}
			},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			if tc.setupMock != nil {
				svc.getTransactionsFunc = tc.setupMock()
			}
			h := NewTransactionHandler(svc)

			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			w := httptest.NewRecorder()

			h.GetTransactions(w, req)

			if w.Code != tc.wantStatus {
				t.Errorf("GetTransactions() status = %v, want %v", w.Code, tc.wantStatus)
			}
		})
	}
}

func TestNewTransactionResponse_MapsDomainFields(t *testing.T) {
	domainTx := domain.Transaction{
		UserID: 1,
		Type:   "bet",
		Amount: 1050,
	}
	resp := NewTransactionResponse(domainTx)
	if resp.UserID != 1 || resp.Amount != 1050 || resp.TransactionType != "bet" {
		t.Errorf("Mapping failed: %+v", resp)
	}
}

func TestTransactionHandler_GetTransactions_JSONContract(t *testing.T) {
	svc := &mockService{
		getTransactionsFunc: func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
			return []domain.Transaction{
				{
					UserID:    2,
					Type:      domain.TransactionTypeWin,
					Amount:    2500,
					Timestamp: time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC),
				},
				{
					UserID:    1,
					Type:      domain.TransactionTypeBet,
					Amount:    1000,
					Timestamp: time.Date(2026, 3, 28, 11, 0, 0, 0, time.UTC),
				},
			}, nil
		},
	}
	h := NewTransactionHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	w := httptest.NewRecorder()
	h.GetTransactions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GetTransactions() status = %v, want %v", w.Code, http.StatusOK)
	}

	var got []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON array: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("response len = %d, want 2", len(got))
	}

	for i, item := range got {
		if _, ok := item["user_id"].(float64); !ok {
			t.Fatalf("item[%d].user_id is not number: %#v", i, item["user_id"])
		}
		if _, ok := item["amount"].(float64); !ok {
			t.Fatalf("item[%d].amount is not number: %#v", i, item["amount"])
		}
		if _, ok := item["transaction_type"].(string); !ok {
			t.Fatalf("item[%d].transaction_type is not string: %#v", i, item["transaction_type"])
		}
		if _, ok := item["timestamp"].(string); !ok {
			t.Fatalf("item[%d].timestamp is not string: %#v", i, item["timestamp"])
		}
	}

	// Contract: handler preserves service-provided order.
	if got[0]["user_id"].(float64) != 2 || got[1]["user_id"].(float64) != 1 {
		t.Fatalf("response order changed: %#v", got)
	}
}

func TestTransactionHandler_GetTransactions_EmitsMetrics(t *testing.T) {
	t.Run("ok/success_response_emits_counter_duration_and_flush", func(t *testing.T) {
		metrics := newMockHTTPMetricsSink()
		svc := &mockService{
			getTransactionsFunc: func(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
				return []domain.Transaction{{UserID: 1, Type: domain.TransactionTypeBet, Amount: 100}}, nil
			},
		}
		h := NewTransactionHandlerWithMetrics(svc, metrics)

		req := httptest.NewRequest(http.MethodGet, "/transactions?user_id=1", nil)
		w := httptest.NewRecorder()
		h.GetTransactions(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("GetTransactions() status = %d, want %d", w.Code, http.StatusOK)
		}

		successKey := basemetrics.Key(MetricAPIRequestsTotal, basemetrics.Labels{
			"endpoint": "/transactions",
			"method":   http.MethodGet,
			"status":   strconv.Itoa(http.StatusOK),
			"reason":   "ok",
		})
		if metrics.counters[successKey] != 1 {
			t.Fatalf("success counter = %d, want 1", metrics.counters[successKey])
		}

		durationKey := basemetrics.Key(MetricAPIRequestDurationMs, basemetrics.Labels{
			"endpoint": "get_transactions",
			"method":   http.MethodGet,
		})
		if metrics.durations[durationKey] != 1 {
			t.Fatalf("duration observations = %d, want 1", metrics.durations[durationKey])
		}
		if metrics.flushes != 1 {
			t.Fatalf("flush calls = %d, want 1", metrics.flushes)
		}
	})

	t.Run("err/validation_error_emits_bad_request_counter", func(t *testing.T) {
		metrics := newMockHTTPMetricsSink()
		h := NewTransactionHandlerWithMetrics(&mockService{}, metrics)

		req := httptest.NewRequest(http.MethodGet, "/transactions?user_id=-10", nil)
		w := httptest.NewRecorder()
		h.GetTransactions(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("GetTransactions() status = %d, want %d", w.Code, http.StatusBadRequest)
		}

		badRequestKey := basemetrics.Key(MetricAPIRequestsTotal, basemetrics.Labels{
			"endpoint": "/transactions",
			"method":   http.MethodGet,
			"status":   strconv.Itoa(http.StatusBadRequest),
			"reason":   "non_positive_user_id",
		})
		if metrics.counters[badRequestKey] != 1 {
			t.Fatalf("bad request counter = %d, want 1", metrics.counters[badRequestKey])
		}
	})
}
