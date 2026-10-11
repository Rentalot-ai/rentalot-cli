package rentalotcli_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
)

func TestM3xWorkflowAvailabilityFilteredRuns(t *testing.T) {
	t.Parallel()
	for _, empty := range []bool{false, true} {
		name := "populated"
		if empty {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Representative v1 paths exercise the generic transport, not route
			// validation: this checkout vendors component schemas, not OpenAPI paths.
			const path = "/api/v1/workflows/runs"
			items := []any{}
			if !empty {
				items = append(items, map[string]any{
					"id":         "a1000000-0000-4000-8000-000000000001",
					"templateId": "f1000000-0000-4000-8000-000000000001",
					"contactId":  "c1000000-0000-4000-8000-000000000001", "propertyId": nil,
					"status": "completed", "currentStepIndex": 3, "messagesSent": 2,
					"exitReason": "completed", "startedAt": "2026-10-01T12:00:00Z",
					"completedAt": "2026-10-01T12:05:00Z",
				})
			}
			var requests atomic.Int32
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != path || r.URL.Query().Get("page") != "2" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": items, "pagination": map[string]any{"page": 2, "limit": 20, "total": len(items)},
				})
			})
			resp, err := client.Get(context.Background(), path, rentalotcli.QueryParams{"page": "2"})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}
			var got struct {
				Data       []map[string]any `json:"data"`
				Pagination struct {
					Page  int `json:"page"`
					Limit int `json:"limit"`
					Total int `json:"total"`
				} `json:"pagination"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if got.Data == nil || len(got.Data) != len(items) || got.Pagination.Page != 2 || got.Pagination.Limit != 20 || got.Pagination.Total != len(items) {
				t.Fatalf("filtered envelope changed: %+v", got)
			}
			if !empty && (got.Data[0]["id"] != "a1000000-0000-4000-8000-000000000001" || got.Data[0]["status"] != "completed") {
				t.Errorf("available run changed: %v", got.Data[0])
			}
			if got := requests.Load(); got != 1 {
				t.Errorf("requests = %d, want one request without paging or retries", got)
			}
		})
	}
}

func TestM3xWorkflowAvailabilityGatedRunAndTrigger(t *testing.T) {
	t.Parallel()
	// These representative paths are opaque to Client. The contract under test
	// is terminal HTTP errors, not a generated endpoint binding.
	cases := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"run read", http.MethodGet, "/api/v1/workflows/runs/d1000000-0000-4000-8000-000000000001", http.StatusNotFound},
		{"trigger not found", http.MethodPost, "/api/v1/workflows/trigger", http.StatusNotFound},
		{"trigger forbidden", http.MethodPost, "/api/v1/workflows/trigger", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			body := map[string]any{
				"workflowId": "d1000000-0000-4000-8000-000000000001",
				"contactId":  "c1000000-0000-4000-8000-000000000001",
			}
			message := http.StatusText(tc.status)
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, tc.method, tc.path)
				}
				if tc.method == http.MethodPost {
					var got map[string]any
					if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
						t.Errorf("decoding trigger: %v", err)
					}
					if len(got) != len(body) || got["workflowId"] != body["workflowId"] || got["contactId"] != body["contactId"] {
						t.Errorf("trigger payload = %v, want %v", got, body)
					}
				}
				w.Header().Set("Content-Type", "application/problem+json")
				// An incidental retry header must not make a gated read/trigger retryable.
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"type": "https://rentalot.ai/problems/gated", "title": message,
					"status": tc.status, "detail": message,
				})
			})
			var resp *http.Response
			var err error
			if tc.method == http.MethodPost {
				resp, err = client.Post(context.Background(), tc.path, body)
			} else {
				resp, err = client.Get(context.Background(), tc.path, nil)
			}
			if err != nil {
				t.Fatalf("transport failed: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			apiFailure := rentalotcli.DecodeError(resp)
			var apiErr *rentalotcli.APIError
			if !errors.As(apiFailure, &apiErr) {
				t.Fatalf("decoded error = %v, want APIError", apiFailure)
			}
			if resp.StatusCode != tc.status || apiErr.StatusCode != tc.status || apiErr.Message != message {
				t.Errorf("error lost HTTP failure details: response %d, error %+v", resp.StatusCode, apiErr)
			}
			if apiErr.Retryable() || strings.Contains(apiErr.Error(), "retry after") {
				t.Errorf("gated failure offered a retry: %v", apiErr)
			}
			if got := requests.Load(); got != 1 {
				t.Errorf("requests = %d, want one request without retry", got)
			}
		})
	}
}
