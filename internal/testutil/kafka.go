//go:build integration || test

package testutil

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupKafka starts a temporary Kafka container in KRaft mode using Bitnami image.
// It uses a dynamic port mapping strategy to solve the 'advertised listeners' problem on Windows.
func SetupKafka(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()

	// 1. Find a free port on the host to map 1:1 with the container.
	// This ensures Kafka's advertised listener matches the port the test connects to.
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to resolve local address: %v", err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("failed to bind to a free port: %v", err)
	}
	freePort := l.Addr().(*net.TCPAddr).Port
	l.Close()
	portStr := strconv.Itoa(freePort)

	fmt.Printf("🚀 Starting Bitnami Kafka KRaft on host port %s...\n", portStr)

	req := testcontainers.ContainerRequest{
		Image: "public.ecr.aws/bitnami/kafka:3.4",
		// Map the host freePort to the same port inside the container
		ExposedPorts: []string{portStr + ":" + portStr + "/tcp"},
		Env: map[string]string{
			"KAFKA_CFG_NODE_ID":                        "1",
			"KAFKA_CFG_PROCESS_ROLES":                  "controller,broker",
			"KAFKA_CFG_CONTROLLER_QUORUM_VOTERS":       "1@localhost:9093",
			"KAFKA_CFG_LISTENERS":                      "PLAINTEXT://0.0.0.0:" + portStr + ",CONTROLLER://0.0.0.0:9093",
			"KAFKA_CFG_ADVERTISED_LISTENERS":           "PLAINTEXT://localhost:" + portStr,
			"KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP": "CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT",
			"KAFKA_CFG_CONTROLLER_LISTENER_NAMES":      "CONTROLLER",
			"KAFKA_CFG_INTER_BROKER_LISTENER_NAME":     "PLAINTEXT",
			"ALLOW_PLAINTEXT_LISTENER":                 "yes",
		},
		WaitingFor: wait.ForLog("Kafka Server started").WithStartupTimeout(3 * time.Minute),
	}

	kafkaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start Kafka: %v", err)
	}

	broker := "localhost:" + portStr
	fmt.Printf("✅ Test Kafka is ready at %s\n", broker)

	return broker, func() {
		kafkaContainer.Terminate(ctx)
	}
}

// CreateTopicAndWait ensures the topic is created and visible in metadata.
func CreateTopicAndWait(t *testing.T, broker, topic string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := kafkago.DialContext(ctx, "tcp", broker)
	if err != nil {
		t.Fatalf("failed to connect to Kafka: %v", err)
	}
	defer conn.Close()

	err = conn.CreateTopics(kafkago.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		t.Fatalf("failed to create topic %s: %v", topic, err)
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for topic %s metadata", topic)
		case <-ticker.C:
			partitions, err := conn.ReadPartitions(topic)
			if err == nil && len(partitions) > 0 {
				fmt.Printf("✅ Metadata synchronized for topic: %s\n", topic)
				return
			}
		}
	}
}
