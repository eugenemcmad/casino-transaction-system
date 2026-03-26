//go:build integration || test || e2e

package testutil

import "testing"

func onTestcontainerSetupError(t *testing.T, err error) {
	t.Helper()
	if e2eBuild {
		t.Logf("testcontainers setup failed: %v", err)
		t.Fatalf("e2e requires Docker and testcontainers (see log above)")
	}
	t.Skipf("skipping: testcontainers unavailable: %v", err)
}
