package rentalotcli_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
)

func TestIdempotency5ho_ConflictGuidanceAndMetadata(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		message    string
		retryAfter string
		envelope   bool
	}{
		{
			name:       "concurrent problem",
			message:    "An operation with this Idempotency-Key is in progress; retry the same request with the same key.",
			retryAfter: "1",
		},
		{
			name:       "unknown outcome envelope",
			message:    "The operation outcome is unknown; retry with the same Idempotency-Key or reconcile the original operation before proceeding.",
			retryAfter: "1",
			envelope:   true,
		},
		{
			name:    "ordinary conflict",
			message: "A property with this address already exists.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/properties/bulk" {
					t.Errorf("request = %s %s, want POST /api/v1/properties/bulk", r.Method, r.URL.Path)
				}
				if tc.retryAfter != "" {
					w.Header().Set("Retry-After", tc.retryAfter)
				}
				if tc.envelope {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"error": map[string]string{"code": "conflict", "message": tc.message},
					})
					return
				}
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"type": "https://rentalot.ai/problems/conflict", "title": "Conflict",
					"status": http.StatusConflict, "detail": tc.message,
				})
			})

			resp, err := client.Post(context.Background(), "/api/v1/properties/bulk", map[string]any{
				"properties": []map[string]any{{"address": "123 Main St", "monthlyRent": 1500, "bedrooms": 2, "bathrooms": 1}},
			})
			if err != nil {
				t.Fatalf("POST: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			apiErr, ok := rentalotcli.DecodeError(resp).(*rentalotcli.APIError)
			if !ok {
				t.Fatal("expected *APIError")
			}
			wantRetryAfter := 0
			if tc.retryAfter != "" {
				wantRetryAfter = 1
			}
			if apiErr.StatusCode != http.StatusConflict || apiErr.RetryAfter != wantRetryAfter {
				t.Errorf("metadata = (%d, %d), want (409, %d)", apiErr.StatusCode, apiErr.RetryAfter, wantRetryAfter)
			}
			if apiErr.Message != tc.message || !strings.HasSuffix(apiErr.Error(), ": "+tc.message) {
				t.Errorf("error = %q, want original guidance %q", apiErr, tc.message)
			}
			if apiErr.Retryable() {
				t.Error("409 must not become an unconditional timed retry; idempotency retries require the original key")
			}
			if got := requests.Load(); got != 1 {
				t.Errorf("requests = %d, want one submission without automatic retries", got)
			}
		})
	}
}
