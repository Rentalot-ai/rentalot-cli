package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
	"github.com/spf13/cobra"
)

func TestM3xWorkflowAvailabilityFilteredLists(t *testing.T) {
	// These fixtures use the vendored WorkflowTemplate and Session shapes.
	// The production server has already omitted lead qualification records.
	resources := []struct {
		name   string
		path   string
		newCmd func() *cobra.Command
		item   map[string]any
		filter string
	}{
		{
			name: "workflows", path: "/api/v1/workflows", newCmd: workflowsListCmd,
			filter: "isActive=true",
			item: map[string]any{
				"id":   "f1000000-0000-4000-8000-000000000001",
				"name": "Pre-Screening", "slug": "pre-screening",
				"description": "Collect a rental application", "isActive": true,
				"isPublic": true, "createdAt": "2026-10-01T12:00:00Z",
			},
		},
		{
			name: "sessions", path: "/api/v1/sessions", newCmd: sessionsListCmd,
			filter: "status=completed",
			item: map[string]any{
				"id":                 "51000000-0000-4000-8000-000000000001",
				"contactId":          "c1000000-0000-4000-8000-000000000001",
				"workflowTemplateId": "f1000000-0000-4000-8000-000000000001",
				"submission":         map[string]any{"budget": 1800},
				"status":             "completed", "reviewStatus": "pending_review",
				"reviewNotes": nil, "reviewedAt": nil,
				"createdAt": "2026-10-01T12:00:00Z", "updatedAt": "2026-10-01T12:05:00Z",
				"expiresAt": "2026-10-31T12:00:00Z",
			},
		},
	}
	for _, resource := range resources {
		for _, empty := range []bool{false, true} {
			for _, asJSON := range []bool{false, true} {
				name := resource.name + "/populated/table"
				if empty {
					name = resource.name + "/empty/table"
				}
				if asJSON {
					name = strings.TrimSuffix(name, "table") + "json"
				}
				t.Run(name, func(t *testing.T) {
					items := []any{}
					if !empty {
						items = append(items, resource.item)
					}
					body := map[string]any{
						"data":       items,
						"pagination": map[string]any{"page": 2, "limit": 20, "total": len(items)},
					}
					var requests atomic.Int32
					handler := func(w http.ResponseWriter, r *http.Request) {
						requests.Add(1)
						if r.Method != http.MethodGet || r.URL.Path != resource.path {
							t.Errorf("request = %s %s, want GET %s", r.Method, r.URL.Path, resource.path)
						}
						filter := strings.SplitN(resource.filter, "=", 2)
						for key, want := range map[string]string{"page": "2", "limit": "20", filter[0]: filter[1]} {
							if got := r.URL.Query().Get(key); got != want {
								t.Errorf("query %s = %q, want %q", key, got, want)
							}
						}
						w.Header().Set("Content-Type", "application/json")
						_ = json.NewEncoder(w).Encode(body)
					}
					execute := executeCommandTable
					if asJSON {
						execute = executeCommand
					}
					out, err := execute(t, handler, resource.newCmd(), "--page", "2", "--limit", "20", "--filter", resource.filter)
					if err != nil {
						t.Fatalf("filtered list failed: %v", err)
					}
					if got := requests.Load(); got != 1 {
						t.Errorf("requests = %d, want one request without paging or retries", got)
					}
					if asJSON {
						var got, want any
						if err := json.Unmarshal([]byte(out), &got); err != nil {
							t.Fatalf("output is not JSON: %v; output = %q", err, out)
						}
						encoded, err := json.Marshal(body)
						if err != nil {
							t.Fatal(err)
						}
						if err := json.Unmarshal(encoded, &want); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(got, want) {
							t.Errorf("response envelope changed: got %v, want %v", got, want)
						}
					} else {
						lines := strings.Split(strings.TrimSpace(out), "\n")
						if len(lines) != 2+len(items) || !strings.HasPrefix(lines[0], "ID") {
							t.Errorf("unexpected table rows: %q", out)
						}
						if !empty && !strings.Contains(out, resource.item["id"].(string)) {
							t.Errorf("available record missing from table: %q", out)
						}
					}
				})
			}
		}
	}
}

func TestM3xWorkflowAvailabilityGatedReads(t *testing.T) {
	const gatedID = "d1000000-0000-4000-8000-000000000001"
	resources := []struct {
		name   string
		path   string
		newCmd func() *cobra.Command
	}{
		{"workflow", "/api/v1/workflows/" + gatedID, workflowsGetCmd},
		{"session", "/api/v1/sessions/" + gatedID, sessionsGetCmd},
	}
	for _, resource := range resources {
		for _, problem := range []bool{false, true} {
			name := resource.name + "/envelope"
			if problem {
				name = resource.name + "/problem"
			}
			t.Run(name, func(t *testing.T) {
				message := resource.name + " not found"
				code := "not_found"
				body := map[string]any{"error": map[string]any{"code": code, "message": message}}
				if problem {
					code = "https://rentalot.ai/problems/not-found"
					body = map[string]any{"type": code, "title": "Not Found", "status": 404, "detail": message}
				}
				var requests atomic.Int32
				handler := func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Method != http.MethodGet || r.URL.Path != resource.path {
						t.Errorf("request = %s %s, want GET %s", r.Method, r.URL.Path, resource.path)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(body)
				}
				cmd := resource.newCmd()
				cmd.SilenceErrors = true
				cmd.SilenceUsage = true
				out, err := executeCommand(t, handler, cmd, gatedID)
				var apiErr *rentalotcli.APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("error = %v, want wrapped APIError", err)
				}
				if apiErr.StatusCode != http.StatusNotFound || apiErr.Code != code || apiErr.Message != message {
					t.Errorf("API error lost server details: %+v", apiErr)
				}
				if !strings.HasPrefix(err.Error(), "getting "+resource.name+": ") {
					t.Errorf("error missing operation context: %v", err)
				}
				if apiErr.Retryable() {
					t.Error("gated 404 must not be retryable")
				}
				if out != "" {
					t.Errorf("failed read printed resource output: %q", out)
				}
				if got := requests.Load(); got != 1 {
					t.Errorf("requests = %d, want one request without retry", got)
				}
			})
		}
	}
}
