//go:build integration

package kafka

import (
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/internal/testutil"
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type mockServiceIntegration struct {
	processedChan chan domain.Transaction
}

func (m *mockServiceIntegration) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	m.processedChan <- t
	return nil
}

func (m *mockServiceIntegration) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	return nil, nil
}

func TestKafkaConsumer_IntegrationFlow(t *testing.T) {
	// 1. Setup Infrastructure
	broker, cleanup := testutil.SetupKafka(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	topic := "component-test-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	testutil.CreateTopicAndWait(t, broker, topic)

	mockSvc := &mockServiceIntegration{
		processedChan: make(chan domain.Transaction, 1),
	}

	// 2. Start Consumer
	consumer := NewConsumer([]string{broker}, topic, "test-group-"+topic, mockSvc, config.Kafka{})
	go func() {
		_ = consumer.Start(ctx)
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
	defer writer.Close()

	testTx := TransactionDTO{
		UserID:    555,
		Type:      domain.TransactionTypeBet,
		Amount:    "1050",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(testTx)
	if err := writer.WriteMessages(ctx, kafkago.Message{Value: payload}); err != nil {
		t.Fatalf("failed to write message: %v", err)
	}

	// 4. Assert
	select {
	case got := <-mockSvc.processedChan:
		if got.UserID != 555 || got.Amount != 1050 {
			t.Errorf("data mismatch: %+v", got)
		}
		t.Log("consumer integration test successful")
	case <-time.After(20 * time.Second):
		t.Error("timeout: consumer failed to process message")
	}
}
