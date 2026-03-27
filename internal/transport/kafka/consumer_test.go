package kafka

import (
	"casino-transaction-system/internal/domain"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func TestTransactionDTO_ToDomain(t *testing.T) {
	tests := []struct {
		name string
		dto  TransactionDTO
		want domain.Transaction
	}{
		{
			name: "Valid Conversion",
			dto: TransactionDTO{
				UserID:    1,
				Type:      "bet",
				Amount:    10.5,
				Timestamp: "2023-10-27T15:00:00Z",
			},
			want: domain.Transaction{
				UserID: 1,
				Type:   "bet",
				Amount: 10.5,
			},
		},
		{
			name: "Invalid Timestamp - returns zero time",
			dto: TransactionDTO{
				UserID:    1,
				Type:      "win",
				Amount:    100,
				Timestamp: "invalid-date",
			},
			want: domain.Transaction{
				UserID: 1,
				Type:   "win",
				Amount: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dto.ToDomain()
			if got.UserID != tt.want.UserID || got.Type != tt.want.Type || got.Amount != tt.want.Amount {
				t.Errorf("ToDomain() = %+v, want %+v", got, tt.want)
			}
			if tt.dto.Timestamp == "invalid-date" && !got.Timestamp.IsZero() {
				t.Error("Expected zero timestamp for invalid input")
			}
		})
	}
}

func TestNewConsumer(t *testing.T) {
	svc := &mockSvc{}
	c := NewConsumer([]string{"127.0.0.1:9092"}, "test-topic", "test-group", svc)
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

func TestConsumerStart_StopsAndClosesOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &mockReader{}
	svc := &mockSvc{}
	c := &Consumer{reader: reader, svc: svc}

	time.AfterFunc(20*time.Millisecond, cancel)

	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start() returned unexpected error: %v", err)
	}
	if !reader.closed {
		t.Fatal("reader.Close() was not called")
	}
	if svc.calls != 0 {
		t.Fatalf("expected no service calls, got %d", svc.calls)
	}
}

func TestConsumerStart_ValidMessageCallsService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := &mockReader{
		messages: []kafkago.Message{
			{
				Value: []byte(`{"user_id":11,"transaction_type":"bet","amount":12.5,"timestamp":"2026-03-27T10:00:00Z"}`),
			},
		},
	}
	svc := &mockSvc{}
	c := &Consumer{reader: reader, svc: svc}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start() returned unexpected error: %v", err)
	}
	if svc.calls != 1 {
		t.Fatalf("expected 1 service call, got %d", svc.calls)
	}
	if svc.lastTx.UserID != 11 || svc.lastTx.Amount != 12.5 || svc.lastTx.Type != domain.TransactionTypeBet {
		t.Fatalf("unexpected transaction passed to service: %+v", svc.lastTx)
	}
}

func TestConsumerStart_InvalidPayloadDoesNotCallService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := &mockReader{
		messages: []kafkago.Message{
			{Value: []byte(`{"user_id":11,"transaction_type":"bet","amount":`)},
			{Value: []byte(`{"user_id":11,"transaction_type":"bet","amount":0}`)},
		},
	}
	svc := &mockSvc{}
	c := &Consumer{reader: reader, svc: svc}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start() returned unexpected error: %v", err)
	}
	if svc.calls != 0 {
		t.Fatalf("expected 0 service calls, got %d", svc.calls)
	}
}

func TestConsumerStart_ServiceErrorIsHandledAndLoopContinues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader := &mockReader{
		messages: []kafkago.Message{
			{
				Value: []byte(`{"user_id":22,"transaction_type":"win","amount":45.0,"timestamp":"2026-03-27T10:00:00Z"}`),
			},
		},
	}
	svc := &mockSvc{returnError: errors.New("db down")}
	c := &Consumer{reader: reader, svc: svc}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start() returned unexpected error: %v", err)
	}
	if svc.calls != 1 {
		t.Fatalf("expected 1 service call, got %d", svc.calls)
	}
}
