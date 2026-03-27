package main

import (
	"os"
	"os/exec"
	"testing"
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

	cmd := exec.Command(os.Args[0], "-test.run=TestMain_ExitsOnInvalidConfig")
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
