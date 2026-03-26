//go:build integration

package kafka

import (
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/internal/testutil"
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type mockIntegrationSvc struct {
	processedChan chan domain.Transaction
	mu            sync.Mutex
	errors        []error
}

func (m *mockIntegrationSvc) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	m.mu.Lock()
	if len(m.errors) > 0 {
		err := m.errors[0]
		m.errors = m.errors[1:]
		m.mu.Unlock()
		return err
	}
	m.mu.Unlock()
	m.processedChan <- t
	return nil
}

func (m *mockIntegrationSvc) RegisterTransactions(ctx context.Context, txs []domain.Transaction) error {
	for _, t := range txs {
		if err := m.RegisterTransaction(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockIntegrationSvc) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	return nil, nil
}

func TestKafkaConsumer_IntegrationFlow(t *testing.T) {
	// 1. Setup Infrastructure
	broker, cleanup := testutil.SetupKafka(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	topic := "component-test-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	dlqTopic := topic + ".dlq"
	testutil.CreateTopicAndWait(t, broker, topic)
	testutil.CreateTopicAndWait(t, broker, dlqTopic)

	mockSvc := &mockIntegrationSvc{
		processedChan: make(chan domain.Transaction, 1),
	}

	// 2. Start Consumer
	consumer := NewConsumer([]string{broker}, topic, "test-group-"+topic, mockSvc, config.Kafka{
		RetryBaseDelayMs:  5,
		RetryJitterMs:     5,
		MaxProcessRetries: 2,
		DLQTopic:          dlqTopic,
	})
	consumerErrCh := make(chan error, 1)
	go func() {
		consumerErrCh <- consumer.Start(ctx)
	}()
	defer func() {
		select {
		case err := <-consumerErrCh:
			if err != nil {
				t.Errorf("consumer.Start() error = %v", err)
			}
		case <-time.After(2 * time.Second):
		}
	}()

	// Give consumer time to join group (standard practice for "LastOffset" consumers).
	time.Sleep(10 * time.Second)

	// 3. Send Message
	t.Log("sending test message to Kafka")
	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(broker),
		Topic:        topic,
		RequiredAcks: kafkago.RequireAll,
	}
	defer func() {
		if err := writer.Close(); err != nil {
			t.Errorf("writer.Close() error = %v", err)
		}
	}()

	testTx := TransactionDTO{
		UserID:    555,
		Type:      domain.TransactionTypeBet,
		Amount:    "1050",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	payload, err := json.Marshal(testTx)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := writer.WriteMessages(ctx, kafkago.Message{Value: payload}); err != nil {
		t.Fatalf("WriteMessages() error = %v", err)
	}

	// 4. Assert
	wantUserID := int64(555)
	wantAmount := int64(1050)
	select {
	case got := <-mockSvc.processedChan:
		if got.UserID != wantUserID {
			t.Errorf("UserID = %d, want %d", got.UserID, wantUserID)
		}
		if got.Amount != wantAmount {
			t.Errorf("Amount = %d, want %d", got.Amount, wantAmount)
		}
		t.Log("consumer integration test successful")
	case <-time.After(20 * time.Second):
		t.Fatal("timeout waiting for RegisterTransaction")
	}
}

func TestKafkaConsumer_RoutesMalformedMessageToDLQ_Integration(t *testing.T) {
	broker, cleanup := testutil.SetupKafka(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	topic := "component-dlq-test-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	dlqTopic := topic + ".dlq"
	testutil.CreateTopicAndWait(t, broker, topic)
	testutil.CreateTopicAndWait(t, broker, dlqTopic)

	mockSvc := &mockIntegrationSvc{
		processedChan: make(chan domain.Transaction, 1),
	}
	consumer := NewConsumer([]string{broker}, topic, "test-group-"+topic, mockSvc, config.Kafka{
		DLQTopic:          dlqTopic,
		MaxProcessRetries: 1,
		RetryBaseDelayMs:  5,
		RetryJitterMs:     5,
	})
	consumerErrCh := make(chan error, 1)
	go func() {
		consumerErrCh <- consumer.Start(ctx)
	}()
	defer func() {
		select {
		case err := <-consumerErrCh:
			if err != nil {
				t.Errorf("consumer.Start() error = %v", err)
			}
		case <-time.After(2 * time.Second):
		}
	}()
	time.Sleep(10 * time.Second)

	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(broker),
		Topic:        topic,
		RequiredAcks: kafkago.RequireAll,
	}
	defer func() {
		if err := writer.Close(); err != nil {
			t.Errorf("writer.Close() error = %v", err)
		}
	}()

	if err := writer.WriteMessages(ctx, kafkago.Message{Value: []byte(`{"user_id":11,"transaction_type":"bet","amount":`)}); err != nil {
		t.Fatalf("WriteMessages() error = %v", err)
	}

	dlqReader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: []string{broker},
		Topic:   dlqTopic,
		GroupID: "dlq-reader-" + topic,
	})
	defer func() {
		if err := dlqReader.Close(); err != nil {
			t.Errorf("dlqReader.Close() error = %v", err)
		}
	}()

	dlqCtx, dlqCancel := context.WithTimeout(ctx, 20*time.Second)
	defer dlqCancel()
	msg, err := dlqReader.ReadMessage(dlqCtx)
	if err != nil {
		t.Fatalf("ReadMessage(DLQ) error = %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		t.Fatalf("json.Unmarshal(dlq payload) error = %v", err)
	}
	if payload["topic"] != topic {
		t.Fatalf("DLQ payload topic = %v, want %s", payload["topic"], topic)
	}
	if payload["error"] == nil {
		t.Fatal("DLQ payload must contain error field")
	}
}

func TestKafkaConsumer_RetriesThenProcesses_Integration(t *testing.T) {
	broker, cleanup := testutil.SetupKafka(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	topic := "component-retry-test-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	dlqTopic := topic + ".dlq"
	testutil.CreateTopicAndWait(t, broker, topic)
	testutil.CreateTopicAndWait(t, broker, dlqTopic)

	mockSvc := &mockIntegrationSvc{
		processedChan: make(chan domain.Transaction, 1),
		errors:        []error{errors.New("temporary db outage")},
	}
	consumer := NewConsumer([]string{broker}, topic, "test-group-"+topic, mockSvc, config.Kafka{
		DLQTopic:          dlqTopic,
		MaxProcessRetries: 2,
		RetryBaseDelayMs:  5,
		RetryJitterMs:     5,
	})
	consumerErrCh := make(chan error, 1)
	go func() {
		consumerErrCh <- consumer.Start(ctx)
	}()
	defer func() {
		select {
		case err := <-consumerErrCh:
			if err != nil {
				t.Errorf("consumer.Start() error = %v", err)
			}
		case <-time.After(2 * time.Second):
		}
	}()
	time.Sleep(10 * time.Second)

	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(broker),
		Topic:        topic,
		RequiredAcks: kafkago.RequireAll,
	}
	defer func() {
		if err := writer.Close(); err != nil {
			t.Errorf("writer.Close() error = %v", err)
		}
	}()

	testTx := TransactionDTO{
		UserID:    777,
		Type:      domain.TransactionTypeWin,
		Amount:    "2100",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	payload, err := json.Marshal(testTx)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := writer.WriteMessages(ctx, kafkago.Message{Value: payload}); err != nil {
		t.Fatalf("WriteMessages() error = %v", err)
	}

	select {
	case got := <-mockSvc.processedChan:
		if got.UserID != 777 || got.Amount != 2100 {
			t.Fatalf("processed transaction = %+v, want user=777 amount=2100", got)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("timeout waiting for retried message processing")
	}
}
