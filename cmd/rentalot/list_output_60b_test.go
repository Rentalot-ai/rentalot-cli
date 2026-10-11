package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
	"github.com/spf13/cobra"
)

type rentalothz60bListCase struct {
	name    string
	command func() *cobra.Command
	item    string
	headers []string
	args    []string
	query   url.Values
}

func rentalothz60bListCases() []rentalothz60bListCase {
	return []rentalothz60bListCase{
		{
			name: "properties", command: propertiesListCmd,
			item:    `{"id":"a824d501-4af2-440d-9cdc-0c5d3f7cef44","title":"Downtown Loft","address":"123 Main St","unitNumber":null,"buildingId":"8a43fcd7-78db-4f9f-9161-f06e56020748","monthlyRent":1800,"bedrooms":0,"bathrooms":1.5,"propertyType":"apartment","status":"active","features":["balcony"]}`,
			headers: []string{"TITLE", "ADDRESS", "UNIT", "MONTHLY RENT"},
			args:    []string{"--property-type", "apartment", "--property-type", "condo", "--bedroom-type", "studio"},
			query:   url.Values{"propertyType": {"apartment", "condo"}, "bedroomType": {"studio"}},
		},
		{
			name: "contacts", command: contactsListCmd,
			item:    `{"id":"0924fe36-46b7-4a84-b695-5c1c028e649b","name":"Ada Lovelace","email":"ada@example.com","phone":null,"role":"tenant","status":"prospect","language":"en","channelPreference":"email"}`,
			headers: []string{"NAME", "EMAIL", "PHONE", "ROLE"},
		},
		{
			name: "showings", command: showingsListCmd,
			item:    `{"id":"c01c5a7c-c448-434e-bc2e-bb910fb7c0b9","propertyId":"a824d501-4af2-440d-9cdc-0c5d3f7cef44","contactId":"0924fe36-46b7-4a84-b695-5c1c028e649b","title":"Loft tour","startTime":"2026-10-12T17:00:00Z","endTime":"2026-10-12T17:30:00Z","timeZone":"America/Los_Angeles","status":"confirmed","notes":"Use side entrance"}`,
			headers: []string{"PROPERTY", "CONTACT", "SCHEDULED AT", "STATUS"},
			args:    []string{"--property-id", "p1", "--contact-id", "c1"},
			query:   url.Values{"property_id": {"p1"}, "contact_id": {"c1"}},
		},
		{
			name: "conversations", command: conversationsListCmd,
			item:    `{"id":"4fdb0fd4-a78f-4147-8ca6-cbd9d2e3b723","contactId":"0924fe36-46b7-4a84-b695-5c1c028e649b","channel":"email","status":"active","lastMessageAt":"2026-10-10T18:00:00Z","summary":"Asked about parking"}`,
			headers: []string{"CONTACT", "CHANNEL", "STATUS", "LAST MESSAGE"},
			args:    []string{"--contact-id", "c1"},
			query:   url.Values{"contact_id": {"c1"}},
		},
	}
}

// Use a fresh command tree to exercise flag parsing without real configuration.
func executeRentalothz60bList(t *testing.T, handler http.HandlerFunc, tc rentalothz60bListCase, args ...string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := rentalotcli.NewClient(rentalotcli.Config{APIKey: "test-key", BaseURL: srv.URL})
	root := &cobra.Command{Use: "rentalot", SilenceErrors: true, SilenceUsage: true}
	previous := jsonOutput
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")
	t.Cleanup(func() { jsonOutput = previous })
	group := &cobra.Command{Use: tc.name}
	group.AddCommand(tc.command())
	root.AddCommand(group)
	root.SetContext(context.WithValue(context.Background(), clientKey, client))
	root.SetArgs(append([]string{tc.name, "list"}, args...))
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	stdout, err := captureStdout(t, root.Execute)
	return buf.String() + stdout, err
}

func TestRentalothz60bListOutput(t *testing.T) {
	modes := []struct {
		name  string
		args  []string
		json  bool
		empty bool
	}{
		{name: "default table"},
		{name: "explicit table", args: []string{"--output", "table"}},
		{name: "json", args: []string{"--output", "json"}, json: true},
		{name: "empty json", args: []string{"--output=json"}, json: true, empty: true},
		{name: "legacy json", args: []string{"--json"}, json: true},
		{name: "legacy json with table", args: []string{"--output", "table", "--json"}, json: true},
	}
	for _, tc := range rentalothz60bListCases() {
		for _, mode := range modes {
			t.Run(tc.name+"/"+mode.name, func(t *testing.T) {
				var item map[string]any
				if err := json.Unmarshal([]byte(tc.item), &item); err != nil {
					t.Fatal(err)
				}
				status := str(item, "status")
				items := tc.item
				if mode.empty {
					items = ""
				}
				body := fmt.Sprintf(`{"data":[%s],"pagination":{"page":2,"limit":20,"total":21,"totalPages":2}}`, items)
				calls := 0
				handler := func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Method != http.MethodGet || r.URL.Path != "/api/v1/"+tc.name {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					}
					want := url.Values{"page": {"2"}, "limit": {"20"}, "status": {status}}
					for key, values := range tc.query {
						want[key] = values
					}
					if got := r.URL.Query(); !reflect.DeepEqual(got, want) {
						t.Errorf("query = %v, want %v", got, want)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprint(w, body)
				}
				args := append([]string{"--limit", "20", "--page", "2", "--filter", "status=" + status}, tc.args...)
				args = append(args, mode.args...)
				out, err := executeRentalothz60bList(t, handler, tc, args...)
				if err != nil {
					t.Fatalf("execute: %v", err)
				}
				if calls != 1 {
					t.Fatalf("HTTP calls = %d, want 1", calls)
				}
				if mode.json {
					var got, want any
					if err := json.Unmarshal([]byte(out), &got); err != nil {
						t.Fatalf("stdout is not a single JSON document: %v\n%s", err, out)
					}
					if err := json.Unmarshal([]byte(body), &want); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, want) {
						t.Errorf("JSON response changed: got %v, want %v", got, want)
					}
				} else {
					for _, header := range append([]string{"ID"}, tc.headers...) {
						if !strings.Contains(out, header) {
							t.Errorf("table missing %q: %s", header, out)
						}
					}
					if !strings.Contains(out, str(item, "id")) || json.Valid([]byte(out)) {
						t.Errorf("expected a table with the returned row: %s", out)
					}
				}
			})
		}
	}
}

func TestRentalothz60bListInvalidOutput(t *testing.T) {
	for _, tc := range rentalothz60bListCases() {
		for _, format := range []string{"yaml", "", "JSON"} {
			t.Run(tc.name+"/"+format, func(t *testing.T) {
				calls := 0
				out, err := executeRentalothz60bList(t, func(w http.ResponseWriter, r *http.Request) {
					calls++
					_, _ = fmt.Fprint(w, `{"data":[]}`)
				}, tc, "--output", format)
				if err == nil || !strings.Contains(err.Error(), "invalid --output") {
					t.Fatalf("error = %v, want invalid output format", err)
				}
				if calls != 0 || out != "" {
					t.Errorf("invalid output made %d requests and printed %q", calls, out)
				}
			})
		}
	}
}

func TestRentalothz60bListJSONErrors(t *testing.T) {
	for _, tc := range rentalothz60bListCases() {
		for _, failure := range []struct {
			name   string
			status int
			body   string
			want   string
		}{
			{"API error", http.StatusUnauthorized, `{"error":{"code":"unauthorized","message":"test key expired"}}`, "test key expired"},
			{"malformed response", http.StatusOK, `{"data":[`, "decoding response"},
		} {
			t.Run(tc.name+"/"+failure.name, func(t *testing.T) {
				out, err := executeRentalothz60bList(t, func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(failure.status)
					_, _ = fmt.Fprint(w, failure.body)
				}, tc, "--output", "json")
				if err == nil || !strings.Contains(err.Error(), failure.want) {
					t.Fatalf("error = %v, want %q", err, failure.want)
				}
				if out != "" {
					t.Errorf("failed command printed success output: %q", out)
				}
			})
		}
	}
}
