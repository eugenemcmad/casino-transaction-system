package kafka

import (
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/domain"
	"context"
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
}

func (m *mockReader) ReadMessage(ctx context.Context) (kafkago.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.idx < len(m.messages) {
		msg := m.messages[m.idx]
		m.idx++
		return msg, nil
	}

	if m.idx-len(m.messages) < len(m.errors) {
		err := m.errors[m.idx-len(m.messages)]
		m.idx++
		return kafkago.Message{}, err
	}

	<-ctx.Done()
	return kafkago.Message{}, ctx.Err()
}

func (m *mockReader) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

type mockSvc struct {
	mu          sync.Mutex
	calls       int
	returnError error
	lastTx      domain.Transaction
}

func (m *mockSvc) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	m.lastTx = t
	return m.returnError
}

func (m *mockSvc) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	return nil, nil
}

func TestConsumerStart_ProcessesMessagesAndHandlesErrors(t *testing.T) {
	dbDownErr := errors.New("db down")
	cases := []struct {
		name       string
		messages   []kafkago.Message
		serviceErr error
		wantCalls  int
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
			wantCalls: 1,
			wantTx: &domain.Transaction{
				UserID: 11,
				Type:   domain.TransactionTypeBet,
				Amount: 1250,
			},
		},
		{
			name: "err/invalid_payload_does_not_call_service",
			messages: []kafkago.Message{
				{Value: []byte(`{"user_id":11,"transaction_type":"bet","amount":`)},
				{Value: []byte(`{"user_id":11,"transaction_type":"bet","amount":0}`)},
			},
			wantCalls: 0,
		},
		{
			name: "err/service_error_is_handled_and_loop_continues",
			messages: []kafkago.Message{
				{Value: []byte(`{"user_id":22,"transaction_type":"win","amount":4500,"timestamp":"2026-03-27T10:00:00Z"}`)},
			},
			serviceErr: dbDownErr,
			wantCalls:  1,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			reader := &mockReader{messages: tc.messages}
			svc := &mockSvc{returnError: tc.serviceErr}
			c := &Consumer{reader: reader, svc: svc}

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
			if tc.wantTx != nil {
				if svc.lastTx.UserID != tc.wantTx.UserID || svc.lastTx.Amount != tc.wantTx.Amount || svc.lastTx.Type != tc.wantTx.Type {
					t.Fatalf("unexpected transaction passed to service: %+v", svc.lastTx)
				}
			}
		})
	}
}
