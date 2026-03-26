//go:build integration

package e2e

import (
	"bytes"
	"casino-transaction-system/internal/app"
	"casino-transaction-system/internal/config"
	"casino-transaction-system/internal/domain"
	transport "casino-transaction-system/internal/transport/http"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestTransactionFlow_E2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. Start Postgres
	fmt.Println("🚀 Starting PostgreSQL container...")
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("casino"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("pass"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres: %v", err)
	}
	defer pgContainer.Terminate(ctx)

	pgConnStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")

	// 2. Setup Database Schema
	db, err := sql.Open("postgres", pgConnStr)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE transactions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			type VARCHAR(10) NOT NULL,
			amount NUMERIC(15, 2) NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			UNIQUE NULLS NOT DISTINCT (user_id, type, amount, timestamp)
		);
	`)
	if err != nil {
		t.Fatalf("failed to setup schema: %v", err)
	}
	db.Close()

	// 3. Start Kafka (using GenericContainer for Bitnami KRaft)
	fmt.Println("🚀 Starting Kafka container (KRaft mode)...")
	kafkaReq := testcontainers.ContainerRequest{
		Image:        "public.ecr.aws/bitnami/kafka:3.4",
		ExposedPorts: []string{"9092/tcp"},
		Env: map[string]string{
			"KAFKA_CFG_NODE_ID":                        "1",
			"KAFKA_CFG_PROCESS_ROLES":                  "controller,broker",
			"KAFKA_CFG_CONTROLLER_QUORUM_VOTERS":       "1@127.0.0.1:9093",
			"KAFKA_CFG_LISTENERS":                      "PLAINTEXT://0.0.0.0:9092,CONTROLLER://0.0.0.0:9093",
			"KAFKA_CFG_ADVERTISED_LISTENERS":           "PLAINTEXT://127.0.0.1:9092",
			"KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP": "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT",
			"KAFKA_CFG_CONTROLLER_LISTENER_NAMES":      "CONTROLLER",
			"KAFKA_CFG_INTER_BROKER_LISTENER_NAME":     "PLAINTEXT",
			"ALLOW_PLAINTEXT_LISTENER":                 "yes",
		},
		WaitingFor: wait.ForLog("Kafka Server started").WithStartupTimeout(3 * time.Minute),
	}

	kafkaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: kafkaReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start kafka: %v", err)
	}
	defer kafkaContainer.Terminate(ctx)

	kHost, _ := kafkaContainer.Host(ctx)
	kPort, _ := kafkaContainer.MappedPort(ctx, "9092")
	brokers := []string{net.JoinHostPort(kHost, kPort.Port())}
	fmt.Printf("✅ Kafka running at %v\n", brokers)

	// 4. Configure and Start Apps (API & Processor)
	fmt.Println("🚀 Starting Application services...")
	cfg := &config.Config{}
	cfg.Postgres.URL = pgConnStr
	cfg.Kafka.Brokers = brokers
	cfg.Kafka.Topic = "e2e-test-topic"
	cfg.Kafka.GroupID = "e2e-test-group"
	cfg.HTTP.Port = "8081"
	cfg.App.Name = "e2e-test-app"
	cfg.App.Version = "1.0.0"

	// Reset config singleton to ensure our new cfg is used (if apps use global config)
	config.ResetConfig()

	apiApp := app.NewApiApp(cfg)
	processorApp, err := app.NewProcessorApp(cfg)
	if err != nil {
		t.Fatalf("failed to init processor: %v", err)
	}

	appCtx, appStop := context.WithCancel(ctx)
	defer appStop()

	go apiApp.Run(appCtx)
	go func() {
		if err := processorApp.Run(appCtx); err != nil {
			fmt.Printf("Processor stopped: %v\n", err)
		}
	}()

	// Wait for apps to initialize
	time.Sleep(5 * time.Second)

	// 5. Execute End-to-End Flow
	testUserID := int64(time.Now().UnixNano() % 1000000)
	testAmount := 99.99
	apiURL := "http://127.0.0.1:8081"

	txReq := map[string]interface{}{
		"user_id":          testUserID,
		"transaction_type": "bet",
		"amount":           testAmount,
		"timestamp":        time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(txReq)

	// 5.1 POST Transaction via API
	resp, err := http.Post(apiURL+"/transactions", "application/json", bytes.NewBuffer(body))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST transaction failed: %v, status: %v", err, resp.StatusCode)
	}

	// 5.2 POLLING for result via Query API
	fmt.Println("⏳ Polling Query API for processed transaction...")
	var transactions []transport.TransactionResponse
	success := false
	for i := 0; i < 20; i++ { // Increased polling time for Kafka latency
		getResp, err := http.Get(fmt.Sprintf("%s/transactions?user_id=%d", apiURL, testUserID))
		if err == nil && getResp.StatusCode == http.StatusOK {
			b, _ := io.ReadAll(getResp.Body)
			json.Unmarshal(b, &transactions)
			getResp.Body.Close()
			if len(transactions) > 0 {
				success = true
				break
			}
		}
		time.Sleep(1 * time.Second)
	}

	if !success {
		t.Fatal("❌ E2E Flow failed: transaction did not appear in database")
	}

	// 5.3 Final Validation
	got := transactions[0]
	if got.UserID != testUserID || got.Amount != testAmount || got.TransactionType != domain.TransactionTypeBet {
		t.Errorf("❌ E2E Flow data mismatch: got %+v, want userID %d, amount %v", got, testUserID, testAmount)
	}
	fmt.Println("✅ E2E Flow successful!")
}
