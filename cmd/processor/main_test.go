package main

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestMain_ExitsOnInvalidConfig(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		main()
		return
	}

	tmpFile, err := os.CreateTemp("", "bad-config-*.yaml")
	if err != nil {
		t.Fatalf("CreateTemp() error: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(":\ninvalid: ["); err != nil {
		t.Fatalf("WriteString() error: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestMain_ExitsOnInvalidConfig")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"CONFIG_PATH="+tmpFile.Name(),
	)

	err = cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code, got nil error")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("exit code = %d, want 1", exitErr.ExitCode())
	}
}

func TestMain_ExitsOnProcessorAppInitError(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		main()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestMain_ExitsOnProcessorAppInitError")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"CONFIG_PATH=does-not-exist.yaml",
		"APP_NAME=test-app",
		"APP_VERSION=1.0.0",
		"PG_URL=postgres://user:pass@127.0.0.1:1/testdb?sslmode=disable",
		"KAFKA_BROKERS=localhost:9092",
		"KAFKA_TOPIC=test-topic",
	)

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code, got nil error")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("exit code = %d, want 1", exitErr.ExitCode())
	}
}
