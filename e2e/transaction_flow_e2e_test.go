//go:build e2e

package e2e

import (
	"casino-transaction-system/internal/bootstrap"
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
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	_ "github.com/lib/pq"
	kafkago "github.com/segmentio/kafka-go"
)

func pickFreeTCPPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer ln.Close()
	return strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

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
	cfg.HTTP.Port = pickFreeTCPPort(t)
	cfg.App.Name = "e2e-test-app"
	cfg.App.Version = "1.0.0"

	config.ResetConfig()

	// 3. Start System Components
	apiApp, err := bootstrap.NewApiApp(cfg)
	if err != nil {
		t.Fatalf("bootstrap.NewApiApp() error = %v", err)
	}
	processorApp, err := bootstrap.NewProcessorApp(cfg)
	if err != nil {
		t.Fatalf("bootstrap.NewProcessorApp() error = %v", err)
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
	testAmount := int64(12345)

	t.Log("seeding transaction message to Kafka")
	writer := &kafkago.Writer{Addr: kafkago.TCP(broker), Topic: topic, RequiredAcks: kafkago.RequireAll}
	defer writer.Close()

	testTx := kafka.TransactionDTO{
		UserID:    testUserID,
		Type:      domain.TransactionTypeWin,
		Amount:    strconv.FormatInt(testAmount, 10),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(testTx)
	if err := writer.WriteMessages(ctx, kafkago.Message{Value: payload}); err != nil {
		t.Fatalf("WriteMessages() error = %v", err)
	}
	t.Log("message seeded successfully")

	// 5. Assert: Verify result via Query API (Polling)
	t.Log("polling query API for result")
	apiURL := "http://127.0.0.1:" + cfg.HTTP.Port
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
		t.Fatal("poll API: no transaction found within deadline")
	}

	// 6. Data Validation
	got := transactions[0]
	if got.UserID != testUserID {
		t.Errorf("UserID = %d, want %d", got.UserID, testUserID)
	}
	if got.Amount != testAmount {
		t.Errorf("Amount = %d, want %d", got.Amount, testAmount)
	}
	t.Log("e2e transaction flow verified successfully")
}
