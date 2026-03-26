//go:build integration

package e2e

import (
	"bytes"
	"casino-transaction-system/internal/domain"
	transport "casino-transaction-system/internal/transport/http" // Alias to avoid conflict with net/http
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	apiURL = "http://127.0.0.1:8080"
)

func TestTransactionFlow_E2E(t *testing.T) {
	// 1. Прямая проверка API Health
	resp, err := http.Get(apiURL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skip("API not available, skipping E2E test. Run 'make docker-up' first.")
	}

	testUserID := int64(time.Now().UnixNano() % 1000000)

	// 2. Создаем транзакцию через POST (Test endpoint)
	txReq := map[string]interface{}{
		"user_id":          testUserID,
		"transaction_type": "bet",
		"amount":           55.55,
		"timestamp":        time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(txReq)

	postResp, err := http.Post(apiURL+"/transactions", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d", postResp.StatusCode)
	}

	// 3. Проверяем асинхронную обработку через GET Query API
	var transactionsResp []transport.TransactionResponse
	success := false

	for i := 0; i < 15; i++ { // Increased attempts for Kafka/DB latency
		getResp, err := http.Get(fmt.Sprintf("%s/transactions?user_id=%d", apiURL, testUserID))
		if err == nil && getResp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(getResp.Body)
			json.Unmarshal(body, &transactionsResp)
			getResp.Body.Close()

			if len(transactionsResp) > 0 {
				success = true
				break
			}
		}
		time.Sleep(1 * time.Second)
	}

	if !success {
		t.Fatal("Transaction did not appear in database via Query API after 15 seconds")
	}

	// 4. Валидация данных
	got := transactionsResp[0]
	if got.UserID != testUserID || got.TransactionType != domain.TransactionTypeBet || got.Amount != 55.55 {
		t.Errorf("Data mismatch: got %+v", got)
	}
}
