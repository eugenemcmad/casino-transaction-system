//go:build integration

package kafka

import (
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

// Mock service implementation for Kafka integration test
type integrationMockService struct {
	processedChan chan domain.Transaction
}

func (m *integrationMockService) RegisterTransaction(ctx context.Context, t domain.Transaction) error {
	m.processedChan <- t
	return nil
}

func (m *integrationMockService) GetTransactions(ctx context.Context, userID int64, tType *domain.TransactionType) ([]domain.Transaction, error) {
	return nil, nil
}

var (
	testKafkaBroker = "127.0.0.1:9094" // Use IPv4 explicitly to avoid Windows localhost issues
)

// waitForKafka attempts to connect to Kafka until it's ready or timeout
func waitForKafka(broker string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("Kafka broker %s not ready within %v", broker, timeout)
		case <-ticker.C:
			conn, err := kafka.DialContext(ctx, "tcp", broker)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}

func TestMain(m *testing.M) {
	// Reset config singleton for each test package
	config.ResetConfig()

	// Set config path relative to test directory
	if os.Getenv("CONFIG_PATH") == "" {
		os.Setenv("CONFIG_PATH", "../../../config.yaml")
	}

	// Load config to get Kafka broker list
	cfg, err := config.NewConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	if len(cfg.Kafka.Brokers) > 0 {
		testKafkaBroker = cfg.Kafka.Brokers[0]
	}

	fmt.Printf("Waiting for Kafka broker %s to be ready...\n", testKafkaBroker)
	if err := waitForKafka(testKafkaBroker, 30*time.Second); err != nil {
		fmt.Printf("Kafka not ready: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Kafka is ready.")

	code := m.Run()
	os.Exit(code)
}

func TestKafkaConsumer_Integration(t *testing.T) {
	topic := "test-transactions-" + fmt.Sprint(time.Now().UnixNano()) // Unique topic for the test
	groupID := "test-group-" + fmt.Sprint(time.Now().UnixNano())      // Unique group ID for the test

	// Context for Kafka operations
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Explicitly create the topic
	conn, err := kafka.DialContext(ctx, "tcp", testKafkaBroker)
	if err != nil {
		t.Fatalf("Failed to dial Kafka for topic creation: %v", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		t.Fatalf("Failed to get Kafka controller: %v", err)
	}
	controllerConn, err := kafka.DialContext(ctx, "tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		t.Fatalf("Failed to dial Kafka controller: %v", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		t.Fatalf("Failed to create topic %s: %v", topic, err)
	}
	t.Logf("Topic %s created successfully.", topic)

	// 2. Create a producer to send a test message
	writer := &kafka.Writer{
		Addr:     kafka.TCP(testKafkaBroker),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// 3. Setup mock service and consumer
	mockSvc := &integrationMockService{
		processedChan: make(chan domain.Transaction, 1),
	}
	consumer := NewConsumer([]string{testKafkaBroker}, topic, groupID, mockSvc)

	// 4. Start consumer in a goroutine
	go func() {
		_ = consumer.Start(ctx)
	}()

	// Give consumer some time to connect and subscribe
	time.Sleep(3 * time.Second)

	// 5. Send a test message
	testTx := TransactionDTO{
		UserID:    999,
		Type:      domain.TransactionTypeBet,
		Amount:    123.45,
		Timestamp: "2023-10-27T15:00:00Z",
	}
	payload, _ := json.Marshal(testTx)

	err = writer.WriteMessages(ctx, kafka.Message{
		Value: payload,
	})
	if err != nil {
		t.Fatalf("Failed to write message to Kafka: %v", err)
	}

	// 6. Wait for service to receive the message (increased timeout for Kafka)
	select {
	case got := <-mockSvc.processedChan:
		if got.UserID != 999 || got.Amount != 123.45 {
			t.Errorf("Consumer processed wrong data: %+v", got)
		}
	case <-time.After(20 * time.Second):
		t.Error("Timed out waiting for consumer to process message")
	}
}
