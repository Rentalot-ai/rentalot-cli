package e2e_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestIdempotency5ho_BinaryReportsGuidanceWithoutResubmitting(t *testing.T) {
	t.Parallel()
	for _, message := range []string{
		"An operation with this Idempotency-Key is in progress; retry the same request with the same key.",
		"The operation outcome is unknown; retry with the same Idempotency-Key or reconcile the original operation before proceeding.",
	} {
		for _, jsonMode := range []bool{false, true} {
			name := "human: " + message
			if jsonMode {
				name = "json: " + message
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				var requests atomic.Int32
				srv := mockAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Method != http.MethodPost || r.URL.Path != "/api/v1/properties/bulk" {
						t.Errorf("request = %s %s, want only the initial bulk submission", r.Method, r.URL.Path)
					}
					w.Header().Set("Content-Type", "application/problem+json")
					w.Header().Set("Retry-After", "1")
					w.WriteHeader(http.StatusConflict)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"type": "https://rentalot.ai/problems/conflict", "title": "Conflict",
						"status": http.StatusConflict, "detail": message,
					})
				}))
				input := filepath.Join(t.TempDir(), "properties.json")
				if err := os.WriteFile(input, []byte(`[{"address":"123 Main St","monthlyRent":1500,"bedrooms":2,"bathrooms":1}]`), 0o600); err != nil {
					t.Fatal(err)
				}
				args := []string{"bulk-import", "--file", input}
				if jsonMode {
					args = append(args, "--json")
				}
				stdout, stderr, err := runBinary(t, envForMock(srv), args...)
				if err == nil {
					t.Fatal("expected nonzero exit for 409")
				}
				if stdout != "" {
					t.Errorf("stdout = %q, want no successful submission output", stdout)
				}
				if !strings.Contains(stderr, "submitting import:") || !strings.Contains(stderr, message) {
					t.Errorf("stderr = %q, want original same-key/reconciliation guidance", stderr)
				}
				if got := requests.Load(); got != 1 {
					t.Errorf("requests = %d, want no resubmission or polling after 409", got)
				}
			})
		}
	}
}
