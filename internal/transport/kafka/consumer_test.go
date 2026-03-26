package kafka

import (
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/internal/repository"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func TestTransactionDTO_ToDomain(t *testing.T) {
	cases := []struct {
		name string
		dto  TransactionDTO
		want domain.Transaction
	}{
		{
			name: "ok/converts_valid_dto",
			dto: TransactionDTO{
				UserID:    1,
				Type:      "bet",
				Amount:    "1050",
				Timestamp: "2023-10-27T15:00:00Z",
			},
			want: domain.Transaction{
				UserID: 1,
				Type:   "bet",
				Amount: 1050,
			},
		},
		{
			name: "err/invalid_timestamp_returns_zero_time",
			dto: TransactionDTO{
				UserID:    1,
				Type:      "win",
				Amount:    "100",
				Timestamp: "invalid-date",
			},
			want: domain.Transaction{
				UserID: 1,
				Type:   "win",
				Amount: 100,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.dto.ToDomain()
			if err != nil {
				t.Fatalf("ToDomain() error = %v", err)
			}
			if got.UserID != tc.want.UserID || got.Type != tc.want.Type || got.Amount != tc.want.Amount {
				t.Errorf("ToDomain() = %+v, want %+v", got, tc.want)
			}
			if tc.dto.Timestamp == "invalid-date" && !got.Timestamp.IsZero() {
				t.Error("expected zero timestamp for invalid input")
			}
		})
	}
}

func TestNewConsumer_CreatesConsumer(t *testing.T) {
	svc := &mockSvc{}
	c := NewConsumer([]string{"127.0.0.1:9092"}, "test-topic", "test-group", svc, config.Kafka{})
	if c == nil {
		t.Fatal("NewConsumer() returned nil")
	}
	if c.reader == nil {
		t.Fatal("NewConsumer() reader is nil")
	}
}

type mockReader struct {
	mu       sync.Mutex
	messages []kafkago.Message
	errors   []error
	idx      int
	closed   bool
	commits  []kafkago.Message
}

func (m *mockReader) FetchMessage(ctx context.Context) (kafkago.Message, error) {
	m.mu.Lock()
	if m.idx < len(m.messages) {
		msg := m.messages[m.idx]
		m.idx++
		m.mu.Unlock()
		return msg, nil
	}

	if m.idx-len(m.messages) < len(m.errors) {
		err := m.errors[m.idx-len(m.messages)]
		m.idx++
		m.mu.Unlock()
		return kafkago.Message{}, err
	}
	m.mu.Unlock()

	<-ctx.Done()
	return kafkago.Message{}, ctx.Err()
}

func (m *mockReader) CommitMessages(ctx context.Context, msgs ...kafkago.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commits = append(m.commits, msgs...)
	return nil
}

func (m *mockReader) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

type mockWriter struct {
	mu       sync.Mutex
	writes   []kafkago.Message
	closed   bool
	writeErr error
}

func (m *mockWriter) WriteMessages(ctx context.Context, msgs ...kafkago.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.writeErr != nil {
		return m.writeErr
	}
	m.writes = append(m.writes, msgs...)
	return nil
}

func (m *mockWriter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

type mockMetrics struct {
	mu       sync.Mutex
	counters map[string]int64
}

func newMockMetrics() *mockMetrics {
	return &mockMetrics{
		counters: make(map[string]int64),
	}
}

func (m *mockMetrics) IncCounter(name string, labels metricLabels, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[metricKey(name, labels)] += value
}

func (m *mockMetrics) SetGauge(name string, labels metricLabels, value float64) {}

func (m *mockMetrics) ObserveDuration(name string, labels metricLabels, d time.Duration) {}

func (m *mockMetrics) Flush() {}

type mockSvc struct {
	mu          sync.Mutex
	calls       int
	returnError error
	errors      []error
	lastTx      domain.Transaction
}

func (m *mockSvc) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.lastTx = t
	if len(m.errors) > 0 {
		err := m.errors[0]
		m.errors = m.errors[1:]
		return err
	}
	return m.returnError
}

func (m *mockSvc) RegisterTransactions(ctx context.Context, txs []domain.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls += len(txs)
	if len(txs) > 0 {
		m.lastTx = txs[len(txs)-1]
	}
	if len(m.errors) > 0 {
		err := m.errors[0]
		m.errors = m.errors[1:]
		return err
	}
	return m.returnError
}

func (m *mockSvc) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	return nil, nil
}

type blockingSvc struct {
	started chan int64
	release <-chan struct{}
}

func (s *blockingSvc) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	select {
	case s.started <- t.UserID:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-s.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *blockingSvc) RegisterTransactions(ctx context.Context, txs []domain.Transaction) error {
	if len(txs) == 0 {
		return nil
	}
	// Simulate blocking on the first transaction logic
	select {
	case s.started <- txs[0].UserID:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case <-s.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *blockingSvc) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	return nil, nil
}

func TestConsumerStart_ProcessesMessagesAndHandlesErrors(t *testing.T) {
	dbDownErr := repository.ErrDBUnavailable
	cases := []struct {
		name       string
		messages   []kafkago.Message
		serviceErr error
		serviceSeq []error
		cfg        config.Kafka
		wantCalls  int
		wantCommit int
		wantDLQ    int
		wantTx     *domain.Transaction
	}{
		{
			name:      "ok/stops_and_closes_reader_on_context_cancel",
			messages:  nil,
			wantCalls: 0,
		},
		{
			name: "ok/valid_message_calls_service",
			messages: []kafkago.Message{
				{Value: []byte(`{"user_id":11,"transaction_type":"bet","amount":1250,"timestamp":"2026-03-27T10:00:00Z"}`)},
			},
			wantCalls:  1,
			wantCommit: 1,
			wantDLQ:    0,
			wantTx: &domain.Transaction{
				UserID: 11,
				Type:   domain.TransactionTypeBet,
				Amount: 1250,
			},
		},
		{
			name: "ok/retry_then_success_commits_once",
			messages: []kafkago.Message{
				{Value: []byte(`{"user_id":18,"transaction_type":"win","amount":700,"timestamp":"2026-03-27T10:00:00Z"}`)},
			},
			serviceSeq: []error{dbDownErr, nil},
			cfg: config.Kafka{
				MaxProcessRetries: 2,
				RetryBaseDelayMs:  1,
				RetryJitterMs:     1,
			},
			wantCalls:  2,
			wantCommit: 1,
			wantDLQ:    0,
		},
		{
			name: "err/invalid_payload_routes_to_dlq_and_commits",
			messages: []kafkago.Message{
				{Value: []byte(`{"user_id":11,"transaction_type":"bet","amount":`)},
			},
			wantCalls:  0,
			wantCommit: 1,
			wantDLQ:    1,
		},
		{
			name: "err/service_error_exhausts_retries_routes_to_dlq",
			messages: []kafkago.Message{
				{Value: []byte(`{"user_id":22,"transaction_type":"win","amount":4500,"timestamp":"2026-03-27T10:00:00Z"}`)},
			},
			serviceErr: dbDownErr,
			cfg: config.Kafka{
				MaxProcessRetries: 1,
				RetryBaseDelayMs:  1,
				RetryJitterMs:     1,
			},
			wantCalls:  4, // 2 bulk retries + 2 sequential retries
			wantCommit: 1,
			wantDLQ:    1,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			reader := &mockReader{messages: tc.messages}
			writer := &mockWriter{}
			metrics := newMockMetrics()
			svc := &mockSvc{returnError: tc.serviceErr, errors: tc.serviceSeq}
			cfg := normalizeKafkaConfig(tc.cfg)
			cfg.BatchSize = 1
			cfg.BatchFlushIntervalSec = 1

			c := &Consumer{
				reader:  reader,
				writer:  writer,
				svc:     svc,
				cfg:     cfg,
				metrics: metrics,
				now:     time.Now,
			}

			go func() {
				time.Sleep(20 * time.Millisecond)
				cancel()
			}()

			if err := c.Start(ctx); err != nil {
				t.Fatalf("Start() returned unexpected error: %v", err)
			}
			if !reader.closed {
				t.Fatal("reader.Close() was not called")
			}
			if svc.calls != tc.wantCalls {
				t.Fatalf("service calls = %d, want %d", svc.calls, tc.wantCalls)
			}
			if len(reader.commits) != tc.wantCommit {
				t.Fatalf("commit calls = %d, want %d", len(reader.commits), tc.wantCommit)
			}
			if len(writer.writes) != tc.wantDLQ {
				t.Fatalf("dlq writes = %d, want %d", len(writer.writes), tc.wantDLQ)
			}
			if !writer.closed {
				t.Fatal("writer.Close() was not called")
			}
			if tc.wantTx != nil {
				if svc.lastTx.UserID != tc.wantTx.UserID || svc.lastTx.Amount != tc.wantTx.Amount || svc.lastTx.Type != tc.wantTx.Type {
					t.Fatalf("unexpected transaction passed to service: %+v", svc.lastTx)
				}
			}
		})
	}
}

func TestConsumerStart_ProcessesDifferentPartitionsConcurrently(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := &mockReader{
		messages: []kafkago.Message{
			{Partition: 0, Value: []byte(`{"user_id":101,"transaction_type":"bet","amount":100,"timestamp":"2026-03-27T10:00:00Z"}`)},
			{Partition: 1, Value: []byte(`{"user_id":202,"transaction_type":"win","amount":200,"timestamp":"2026-03-27T10:00:00Z"}`)},
		},
	}
	writer := &mockWriter{}
	metrics := newMockMetrics()
	release := make(chan struct{})
	svc := &blockingSvc{
		started: make(chan int64, 2),
		release: release,
	}
	testCfg := normalizeKafkaConfig(config.Kafka{})
	testCfg.BatchSize = 1
	testCfg.BatchFlushIntervalSec = 1

	c := &Consumer{
		reader:  reader,
		writer:  writer,
		svc:     svc,
		cfg:     testCfg,
		metrics: metrics,
		now:     time.Now,
	}

	done := make(chan error, 1)
	go func() {
		done <- c.Start(ctx)
	}()

	seen := map[int64]bool{}
	timeout := time.NewTimer(300 * time.Millisecond)
	defer timeout.Stop()
	for len(seen) < 2 {
		select {
		case uid := <-svc.started:
			seen[uid] = true
		case <-timeout.C:
			t.Fatalf("expected concurrent processing for two partitions, got starts=%v", seen)
		}
	}

	close(release)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start() returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting consumer shutdown")
	}

	if len(reader.commits) != 2 {
		t.Fatalf("commit calls = %d, want 2", len(reader.commits))
	}
}

func TestRouteToDLQ_RedactsPayloadAndAddsFingerprint(t *testing.T) {
	msgValue := []byte(`{"user_id":999,"transaction_type":"bet","amount":7000}`)
	expectedHash := sha256.Sum256(msgValue)
	expectedHashHex := hex.EncodeToString(expectedHash[:])

	writer := &mockWriter{}
	c := &Consumer{
		writer:  writer,
		metrics: newMockMetrics(),
		now:     time.Now,
	}

	msg := kafkago.Message{
		Topic:     "tx-topic",
		Partition: 3,
		Offset:    42,
		Key:       []byte("user-999"),
		Value:     msgValue,
	}

	if err := c.routeToDLQ(context.Background(), msg, errors.New("validation failed")); err != nil {
		t.Fatalf("routeToDLQ() error = %v", err)
	}

	if len(writer.writes) != 1 {
		t.Fatalf("DLQ writes = %d, want 1", len(writer.writes))
	}

	var payload map[string]any
	if err := json.Unmarshal(writer.writes[0].Value, &payload); err != nil {
		t.Fatalf("json.Unmarshal(dlq payload) error = %v", err)
	}

	if got, _ := payload["value"].(string); got != "[REDACTED]" {
		t.Fatalf("DLQ payload value = %q, want [REDACTED]", got)
	}
	if got, _ := payload["value_sha256"].(string); got != expectedHashHex {
		t.Fatalf("DLQ payload value_sha256 = %q, want %q", got, expectedHashHex)
	}

	gotSize, ok := payload["value_size_bytes"].(float64)
	if !ok {
		t.Fatalf("DLQ payload value_size_bytes type = %T, want float64", payload["value_size_bytes"])
	}
	if int(gotSize) != len(msgValue) {
		t.Fatalf("DLQ payload value_size_bytes = %d, want %d", int(gotSize), len(msgValue))
	}
}
