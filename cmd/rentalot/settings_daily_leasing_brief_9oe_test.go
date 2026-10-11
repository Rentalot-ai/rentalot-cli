package main

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSettingsUpdateDailyLeasingBrief9oe(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want map[string]any
	}{
		{
			name: "enable brief",
			args: []string{"--daily-leasing-brief-enabled=true"},
			want: map[string]any{"emailPreferences": map[string]any{"dailyLeasingBriefEnabled": true}},
		},
		{
			name: "disable brief",
			args: []string{"--daily-leasing-brief-enabled=false"},
			want: map[string]any{"emailPreferences": map[string]any{"dailyLeasingBriefEnabled": false}},
		},
		{
			name: "enable no activity brief",
			args: []string{"--daily-leasing-brief-no-activity-enabled=true"},
			want: map[string]any{"emailPreferences": map[string]any{"dailyLeasingBriefNoActivityEnabled": true}},
		},
		{
			name: "disable no activity brief",
			args: []string{"--daily-leasing-brief-no-activity-enabled=false"},
			want: map[string]any{"emailPreferences": map[string]any{"dailyLeasingBriefNoActivityEnabled": false}},
		},
		{
			name: "combine preferences",
			args: []string{"--daily-leasing-brief-enabled", "--daily-leasing-brief-no-activity-enabled=false"},
			want: map[string]any{"emailPreferences": map[string]any{
				"dailyLeasingBriefEnabled":           true,
				"dailyLeasingBriefNoActivityEnabled": false,
			}},
		},
		{
			name: "combine with followups",
			args: []string{"--daily-leasing-brief-enabled=false", "--daily-leasing-brief-no-activity-enabled", "--followup-enabled", "true", "--followup-delay", "48"},
			want: map[string]any{
				"emailPreferences": map[string]any{
					"dailyLeasingBriefEnabled":           false,
					"dailyLeasingBriefNoActivityEnabled": true,
				},
				"followup_enabled":     true,
				"followup_delay_hours": float64(48),
			},
		},
		{
			name: "combine with business name",
			args: []string{"--business-name", "Cedar Rentals", "--daily-leasing-brief-enabled=false"},
			want: map[string]any{
				"businessName":     "Cedar Rentals",
				"emailPreferences": map[string]any{"dailyLeasingBriefEnabled": false},
			},
		},
		{
			name: "business name omits email preferences",
			args: []string{"--business-name", ""},
			want: map[string]any{"businessName": ""},
		},
		{
			name: "followups omit email preferences",
			args: []string{"--followup-enabled", "false", "--followup-delay", "24"},
			want: map[string]any{"followup_enabled": false, "followup_delay_hours": float64(24)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			handler := func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/settings" {
					t.Errorf("request = %s %s, want PATCH /api/v1/settings", r.Method, r.URL.Path)
				}
				var got map[string]any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Errorf("decoding request: %v", err)
					http.Error(w, "invalid JSON", http.StatusBadRequest)
					return
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("request body = %#v, want %#v", got, tc.want)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"emailPreferences":{"marketing":true,"dailyLeasingBriefEnabled":true,"dailyLeasingBriefNoActivityEnabled":false}}}`))
			}
			output, err := executeCommand(t, handler, settingsUpdateCmd(), tc.args...)
			if err != nil {
				t.Fatalf("updating settings: %v", err)
			}
			if calls.Load() != 1 {
				t.Errorf("requests = %d, want 1", calls.Load())
			}
			var response map[string]any
			if err := json.Unmarshal([]byte(output), &response); err != nil {
				t.Fatalf("decoding output: %v", err)
			}
			wantResponse := map[string]any{"data": map[string]any{"emailPreferences": map[string]any{
				"marketing":                          true,
				"dailyLeasingBriefEnabled":           true,
				"dailyLeasingBriefNoActivityEnabled": false,
			}}}
			if !reflect.DeepEqual(response, wantResponse) {
				t.Errorf("output = %#v, want %#v", response, wantResponse)
			}
		})
	}
}

func TestSettingsUpdateDailyLeasingBriefValidation9oe(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{"--daily-leasing-brief-enabled=invalid"},
		{"--daily-leasing-brief-no-activity-enabled=invalid"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var calls atomic.Int32
			handler := func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(http.StatusOK)
			}
			_, err := executeCommand(t, handler, settingsUpdateCmd(), args...)
			if err == nil {
				t.Fatal("expected an error for missing settings or an invalid boolean")
			}
			if len(args) == 0 && (!strings.Contains(err.Error(), "--daily-leasing-brief-enabled") ||
				!strings.Contains(err.Error(), "--daily-leasing-brief-no-activity-enabled")) {
				t.Errorf("missing-settings error omits the new flags: %v", err)
			}
			if calls.Load() != 0 {
				t.Errorf("requests = %d, want 0", calls.Load())
			}
		})
	}
}

func TestSettingsGetDailyLeasingBrief9oe(t *testing.T) {
	data := map[string]any{"data": map[string]any{
		"agentName": "Leasing Assistant",
		"emailPreferences": map[string]any{
			"marketing":                          false,
			"schedulingEmailEnabled":             true,
			"dailyLeasingBriefEnabled":           true,
			"dailyLeasingBriefNoActivityEnabled": false,
		},
	}}
	handler := jsonHandler(t, http.MethodGet, "/api/v1/settings", http.StatusOK, data)
	output, err := executeCommand(t, handler, settingsGetCmd())
	if err != nil {
		t.Fatalf("getting settings: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("decoding output: %v", err)
	}
	if !reflect.DeepEqual(got, data) {
		t.Errorf("output = %#v, want %#v", got, data)
	}
	output, err = executeCommandTable(t, handler, settingsGetCmd())
	if err != nil {
		t.Fatalf("getting settings as a table: %v", err)
	}
	for _, field := range []string{"dailyLeasingBriefEnabled:true", "dailyLeasingBriefNoActivityEnabled:false"} {
		if !strings.Contains(output, field) {
			t.Errorf("table output %q does not contain %q", output, field)
		}
	}
}
