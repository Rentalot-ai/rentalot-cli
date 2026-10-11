package main

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSettingsBusinessNameZV9Update(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		want     map[string]any
		wantName any
	}{
		{
			name:     "set",
			args:     []string{"--business-name", "Cedar & Pine Rentals"},
			want:     map[string]any{"businessName": "Cedar & Pine Rentals"},
			wantName: "Cedar & Pine Rentals",
		},
		{
			name: "clear",
			args: []string{"--business-name", ""},
			want: map[string]any{"businessName": ""},
		},
		{
			name:     "literal null remains a name",
			args:     []string{"--business-name", "null"},
			want:     map[string]any{"businessName": "null"},
			wantName: "null",
		},
		{
			name:     "100 characters",
			args:     []string{"--business-name", strings.Repeat("A", 100)},
			want:     map[string]any{"businessName": strings.Repeat("A", 100)},
			wantName: strings.Repeat("A", 100),
		},
		{
			name:     "unicode characters",
			args:     []string{"--business-name", strings.Repeat("é", 100)},
			want:     map[string]any{"businessName": strings.Repeat("é", 100)},
			wantName: strings.Repeat("é", 100),
		},
		{
			name:     "combine with followups",
			args:     []string{"--business-name", "Cedar Rentals", "--followup-enabled", "false", "--followup-delay", "48"},
			want:     map[string]any{"businessName": "Cedar Rentals", "followup_enabled": false, "followup_delay_hours": float64(48)},
			wantName: "Cedar Rentals",
		},
		{
			name:     "omit when absent",
			args:     []string{"--followup-enabled", "true"},
			want:     map[string]any{"followup_enabled": true},
			wantName: "Existing Rentals",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			handler := func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/settings" {
					t.Errorf("request = %s %s, want PATCH /api/v1/settings", r.Method, r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decoding request: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if !reflect.DeepEqual(body, tt.want) {
					t.Errorf("body = %#v, want %#v", body, tt.want)
				}
				var businessName any = "Existing Rentals"
				if name, exists := body["businessName"]; exists {
					businessName = name
					if name == "" {
						businessName = nil
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"businessName": businessName, "followupEnabled": true})
			}
			out, err := executeCommand(t, handler, settingsUpdateCmd(), tt.args...)
			if err != nil {
				t.Fatalf("updating settings: %v", err)
			}
			if requests.Load() != 1 {
				t.Fatalf("requests = %d, want 1", requests.Load())
			}
			var response map[string]any
			if err := json.Unmarshal([]byte(out), &response); err != nil {
				t.Fatalf("decoding output: %v", err)
			}
			if value, exists := response["businessName"]; !exists || value != tt.wantName {
				t.Errorf("response = %#v, want businessName = %#v", response, tt.wantName)
			}
		})
	}
}

func TestSettingsBusinessNameZV9RejectsLongNamesBeforeHTTP(t *testing.T) {
	for _, name := range []string{strings.Repeat("A", 101), strings.Repeat("é", 101)} {
		t.Run(name, func(t *testing.T) {
			var requests atomic.Int32
			handler := func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusNoContent)
			}
			_, err := executeCommand(t, handler, settingsUpdateCmd(), "--business-name", name)
			if err == nil || !strings.Contains(err.Error(), "--business-name must be at most 100 characters") {
				t.Fatalf("error = %v, want business name length error", err)
			}
			if requests.Load() != 0 {
				t.Errorf("requests = %d, want no HTTP request", requests.Load())
			}
		})
	}
}

func TestSettingsBusinessNameZV9GetJSON(t *testing.T) {
	for _, response := range []map[string]any{
		{"businessName": "Cedar & Pine Rentals", "followupEnabled": true},
		{"businessName": nil, "followupEnabled": true},
		{"followupEnabled": true},
		{"data": map[string]any{"businessName": "Cedar & Pine Rentals", "followupEnabled": true}},
	} {
		handler := jsonHandler(t, http.MethodGet, "/api/v1/settings", http.StatusOK, response)
		out, err := executeCommand(t, handler, settingsGetCmd())
		if err != nil {
			t.Fatalf("getting settings: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("decoding output: %v", err)
		}
		if !reflect.DeepEqual(got, response) {
			t.Errorf("output = %#v, want %#v", got, response)
		}
	}
}

func TestSettingsBusinessNameZV9GetTable(t *testing.T) {
	response := map[string]any{"businessName": "Cedar & Pine Rentals", "followupEnabled": true}
	out, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/settings", http.StatusOK, response), settingsGetCmd())
	if err != nil {
		t.Fatalf("getting settings: %v", err)
	}
	if !strings.Contains(out, "businessName") || !strings.Contains(out, "Cedar & Pine Rentals") {
		t.Errorf("settings table missing business name: %q", out)
	}
}
