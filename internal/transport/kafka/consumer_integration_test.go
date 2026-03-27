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

type mockIntegrationSvc struct {
	processedChan chan domain.Transaction
}

func (m *mockIntegrationSvc) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	m.processedChan <- t
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
	testutil.CreateTopicAndWait(t, broker, topic)

	mockSvc := &mockIntegrationSvc{
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
