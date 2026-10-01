package e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestBinary_Properties_UnitNumberPayload(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		for _, tc := range []struct {
			name    string
			flags   []string
			present bool
			want    any
		}{
			{name: "omitted"},
			{name: "set", flags: []string{"--unit-number", " 4B "}, present: true, want: "4B"},
			{name: "clear", flags: []string{"--unit-number", ""}, present: true},
			{name: "blank", flags: []string{"--unit-number", "   "}, present: true},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				requests := 0
				srv := mockAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decoding request: %v", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					value, present := body["unitNumber"]
					if present != tc.present || value != tc.want {
						t.Errorf("unitNumber = %v (present %v), want %v (present %v)", value, present, tc.want, tc.present)
					}
					if _, present := body["buildingId"]; present {
						t.Error("read-only buildingId must not be sent")
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "p100"}})
				}))
				args := []string{"properties", operation}
				if operation == "create" {
					args = append(args, "--address", "123 Main St Apt 2", "--monthly-rent", "1800", "--bedrooms", "1", "--bathrooms", "1")
				} else {
					args = append(args, "p100")
					if !tc.present {
						args = append(args, "--title", "Updated Loft")
					}
				}
				args = append(args, tc.flags...)
				_, stderr, err := runBinary(t, envForMock(srv), args...)
				if err != nil {
					t.Fatalf("exit error: %v; stderr: %s", err, stderr)
				}
				if requests != 1 {
					t.Errorf("requests = %d, want 1", requests)
				}
			})
		}
	}
}

func TestBinary_PropertiesList_UnitColumns(t *testing.T) {
	srv := mockAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{
			map[string]any{"id": "p1", "address": "123 Main St", "unitNumber": "4B", "buildingId": "b1"},
			map[string]any{"id": "p2", "address": "123 Main St", "unitNumber": "5C", "buildingId": "b1"},
			map[string]any{"id": "p3", "address": "456 Oak St", "unitNumber": nil, "buildingId": nil},
		}})
	}))
	stdout, stderr, err := runBinary(t, envForMock(srv), "properties", "list")
	if err != nil {
		t.Fatalf("exit error: %v; stderr: %s", err, stderr)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected header, separator and three property rows: %s", stdout)
	}
	if !strings.Contains(lines[0], "UNIT") || !strings.Contains(lines[0], "BUILDING ID") {
		t.Fatalf("missing unit/building columns: %s", stdout)
	}
	for i, unit := range []string{"4B", "5C"} {
		if !strings.Contains(lines[i+2], unit) || !strings.Contains(lines[i+2], "b1") {
			t.Errorf("unit %s missing or on wrong property row: %s", unit, lines[i+2])
		}
	}
	if strings.Contains(lines[4], "<nil>") || strings.Contains(lines[4], "null") {
		t.Errorf("null unit/building must render blank: %s", lines[4])
	}
}
