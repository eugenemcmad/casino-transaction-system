//go:build e2e

package e2e

import (
	"casino-transaction-system/internal/app"
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/domain"
	"casino-transaction-system/internal/testutil"
	transport "casino-transaction-system/internal/transport/http"
	"casino-transaction-system/internal/transport/kafka"
	"casino-transaction-system/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	_ "github.com/lib/pq"
	kafkago "github.com/segmentio/kafka-go"
)

// TestTransactionFlow_E2E verifies the entire asynchronous chain:
// Kafka -> Processor -> PostgreSQL -> HTTP API
func TestTransactionFlow_EndToEnd(t *testing.T) {
	// 1. Setup Infrastructure using testutil
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pgConnStr, pgCleanup := testutil.SetupPostgres(t)
	defer pgCleanup()

	broker, kafkaCleanup := testutil.SetupKafka(t)
	defer kafkaCleanup()

	// Use unique IDs for this test run to ensure isolation
	uniqueID := strconv.FormatInt(time.Now().UnixNano(), 10)
	topic := "e2e-topic-" + uniqueID
	testutil.CreateTopicAndWait(t, broker, topic)

	// 2. Configure Application with DEBUG logging for visibility
	logger.SetupLogger("debug")
	cfg := &config.Config{}
	cfg.Postgres.URL = pgConnStr
	cfg.Kafka.Brokers = []string{broker}
	cfg.Kafka.Topic = topic
	cfg.Kafka.GroupID = "e2e-group-" + uniqueID
	cfg.HTTP.Port = "8083"
	cfg.App.Name = "e2e-test-app"
	cfg.App.Version = "1.0.0"

	config.ResetConfig()

	// 3. Start System Components
	apiApp := app.NewApiApp(cfg)
	processorApp, err := app.NewProcessorApp(cfg)
	if err != nil {
		t.Fatalf("failed to initialize processor app: %v", err)
	}

	appCtx, appStop := context.WithCancel(ctx)
	defer appStop()

	go apiApp.Run(appCtx)
	go processorApp.Run(appCtx)

	// Give components time to fully stabilize (consumer group join etc.)
	t.Log("waiting for processor to stabilize")
	time.Sleep(15 * time.Second)

	// 4. Act: Seed message to Kafka
	testUserID := int64(123456789)
	testAmount := 123.45

	t.Log("seeding transaction message to Kafka")
	writer := &kafkago.Writer{Addr: kafkago.TCP(broker), Topic: topic, RequiredAcks: kafkago.RequireAll}
	defer writer.Close()

	testTx := kafka.TransactionDTO{
		UserID:    testUserID,
		Type:      domain.TransactionTypeWin,
		Amount:    testAmount,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(testTx)
	if err := writer.WriteMessages(ctx, kafkago.Message{Value: payload}); err != nil {
		t.Fatalf("failed to send test message: %v", err)
	}
	t.Log("message seeded successfully")

	// 5. Assert: Verify result via Query API (Polling)
	t.Log("polling query API for result")
	apiURL := "http://127.0.0.1:8083"
	var transactions []transport.TransactionResponse
	success := false

	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		getResp, err := http.Get(fmt.Sprintf("%s/transactions?user_id=%d", apiURL, testUserID))
		if err == nil && getResp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(getResp.Body)
			json.Unmarshal(body, &transactions)
			getResp.Body.Close()
			if len(transactions) > 0 {
				success = true
				break
			}
		}
		time.Sleep(1 * time.Second)
	}

	if !success {
		t.Fatal("e2e flow failed: transaction was not found in DB via API")
	}

	// 6. Data Validation
	got := transactions[0]
	if got.UserID != testUserID || got.Amount != testAmount {
		t.Errorf("data mismatch: expected userID %d and amount %v, got %+v", testUserID, testAmount, got)
	}
	t.Log("e2e transaction flow verified successfully")
}
