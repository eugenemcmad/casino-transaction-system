package boundary

import (
	"casino-transaction-system/internal/repository"
	"casino-transaction-system/pkg/money"
	"net/http"
	"testing"
)

func TestClassify_ReturnsStableMappings(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantCode   string
		wantStatus int
	}{
		{
			name:       "repo_not_initialized",
			err:        repository.ErrRepoNotInitialized,
			wantCode:   "repo_not_initialized",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "db_unavailable",
			err:        repository.ErrDBUnavailable,
			wantCode:   "db_unavailable",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "invalid_amount",
			err:        money.ErrInvalidAmount,
			wantCode:   "invalid_amount_format",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.err)
			if got.Code != tc.wantCode || got.HTTPStatus != tc.wantStatus {
				t.Fatalf("Classify() = (%s, %d), want (%s, %d)", got.Code, got.HTTPStatus, tc.wantCode, tc.wantStatus)
			}
		})
	}
}
