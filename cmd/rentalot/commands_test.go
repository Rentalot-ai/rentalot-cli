package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
	"github.com/spf13/cobra"
)

// executeCommand sets up a fake API server, injects the client into context,
// and executes the given cobra command with args.
func executeCommand(t *testing.T, handler http.HandlerFunc, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client := rentalotcli.NewClient(rentalotcli.Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
	})

	// Inject client into context.
	ctx := context.WithValue(context.Background(), clientKey, client)
	cmd.SetContext(ctx)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)

	// Ensure JSON output for commands that support it.
	jsonOutput = true
	t.Cleanup(func() { jsonOutput = false })

	stdout, err := captureStdout(t, cmd.Execute)
	return buf.String() + stdout, err
}

func captureStdout(t *testing.T, run func() error) (string, error) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating stdout pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = writer
	runErr := run()
	_ = writer.Close()
	os.Stdout = original
	data, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if readErr != nil {
		t.Fatalf("reading captured stdout: %v", readErr)
	}
	return string(data), runErr
}

// executeCommandTable is like executeCommand but with jsonOutput=false to exercise table rendering.
func executeCommandTable(t *testing.T, handler http.HandlerFunc, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client := rentalotcli.NewClient(rentalotcli.Config{
		APIKey:  "test-key",
		BaseURL: srv.URL,
	})

	ctx := context.WithValue(context.Background(), clientKey, client)
	cmd.SetContext(ctx)

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)

	jsonOutput = false
	t.Cleanup(func() { jsonOutput = false })

	stdout, err := captureStdout(t, cmd.Execute)
	return buf.String() + stdout, err
}

// jsonHandler returns an HTTP handler that responds with the given JSON body.
func jsonHandler(t *testing.T, wantMethod, wantPath string, statusCode int, body any) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != wantMethod {
			t.Errorf("method = %q, want %q", r.Method, wantMethod)
		}
		if r.URL.Path != wantPath {
			t.Errorf("path = %q, want %q", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(body)
	}
}

func TestContactsListCmd(t *testing.T) {
	data := map[string]any{
		"data": []any{
			map[string]any{"id": "c1", "name": "Alice", "email": "alice@test.com", "phone": "555-0001", "role": "tenant"},
		},
	}
	handler := jsonHandler(t, http.MethodGet, "/api/v1/contacts", http.StatusOK, data)

	_, err := executeCommand(t, handler, contactsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactsGetCmd(t *testing.T) {
	contact := map[string]any{"id": "c1", "name": "Alice"}
	handler := jsonHandler(t, http.MethodGet, "/api/v1/contacts/c1", http.StatusOK, contact)

	_, err := executeCommand(t, handler, contactsGetCmd(), "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactsCreateCmd(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "Alice" {
			t.Errorf("name = %v, want Alice", body["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "c1", "name": "Alice"})
	}

	_, err := executeCommand(t, handler, contactsCreateCmd(), "--name", "Alice", "--email", "alice@test.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactsCreateCmd_UsesCanonicalRoleAndOmitsEmptyFields(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		want := map[string]any{
			"name":   "Ada Lovelace",
			"email":  "ada@example.com",
			"role":   "tenant",
			"status": "prospect",
		}
		if len(body) != len(want) {
			t.Errorf("payload keys = %v, want exactly %v", body, want)
		}
		for key, value := range want {
			if body[key] != value {
				t.Errorf("body[%q] = %v, want %v", key, body[key], value)
			}
		}
		if _, ok := body["type"]; ok {
			t.Error("legacy field type must not be sent")
		}
		if _, ok := body["phone"]; ok {
			t.Error("empty phone must be omitted")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "c1"}})
	}

	_, err := executeCommand(t, handler, contactsCreateCmd(),
		"--name", "Ada Lovelace", "--email", "ada@example.com", "--role", "tenant", "--status", "prospect")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactsCreateCmd_RequiresEmailOrPhone(t *testing.T) {
	hit := false
	handler := func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		w.WriteHeader(http.StatusCreated)
	}

	_, err := executeCommand(t, handler, contactsCreateCmd(), "--name", "No Contact Method")
	if err == nil {
		t.Fatal("expected error when email and phone are both missing")
	}
	if hit {
		t.Fatal("request should be rejected before contacting the API")
	}
	if !strings.Contains(err.Error(), "email") || !strings.Contains(err.Error(), "phone") {
		t.Errorf("error = %q, want email or phone guidance", err)
	}
}

func TestContactsCreateCmd_LegacyTypeMapsToRole(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body["role"] != "tenant" {
			t.Errorf("role = %v, want tenant", body["role"])
		}
		if _, ok := body["type"]; ok {
			t.Error("legacy field type must not be sent")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "c1"}})
	}

	_, err := executeCommand(t, handler, contactsCreateCmd(), "--name", "Legacy Contact", "--email", "legacy@example.com", "--type", "tenant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactsCreateCmd_MissingName(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	_, err := executeCommand(t, handler, contactsCreateCmd())
	if err == nil {
		t.Error("expected error for missing --name")
	}
}

func TestContactsUpdateCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPatch, "/api/v1/contacts/c1", http.StatusOK, map[string]any{"id": "c1"})

	_, err := executeCommand(t, handler, contactsUpdateCmd(), "c1", "--name", "Bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactsUpdateCmd_UsesCanonicalRole(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body["role"] != "landlord" {
			t.Errorf("role = %v, want landlord", body["role"])
		}
		if _, ok := body["type"]; ok {
			t.Error("legacy field type must not be sent")
		}
		if len(body) != 1 {
			t.Errorf("payload = %v, want only role", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "c1"}})
	}

	_, err := executeCommand(t, handler, contactsUpdateCmd(), "c1", "--role", "landlord")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactsListCmd_UsesCanonicalRoleOutput(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{
		"id": "c1", "name": "Ada Lovelace", "email": "ada@example.com",
		"phone": "+15125550101", "role": "tenant",
	}}}
	output, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/contacts", http.StatusOK, data), contactsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"ROLE", "Ada Lovelace", "tenant"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q: %s", want, output)
		}
	}
}

func TestContactsDeleteCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodDelete, "/api/v1/contacts/c1", http.StatusNoContent, nil)

	_, err := executeCommand(t, handler, contactsDeleteCmd(), "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContactsGetCmd_APIError(t *testing.T) {
	handler := jsonHandler(t, http.MethodGet, "/api/v1/contacts/missing", http.StatusNotFound,
		map[string]any{"error": map[string]any{"code": "not_found", "message": "contact not found"}})

	_, err := executeCommand(t, handler, contactsGetCmd(), "missing")
	if err == nil {
		t.Error("expected error for 404 response")
	}
}

func TestContactsListCmd_WithPagination(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("limit = %q, want 5", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page = %q, want 2", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}

	cmd := contactsListCmd()
	_, err := executeCommand(t, handler, cmd, "--limit", "5", "--page", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Properties ---

func TestPropertiesListCmd(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "p1", "title": "Apt A", "monthlyRent": 1500}}}
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/properties", http.StatusOK, data), propertiesListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesListCmd_EncodesRepeatedPropertyTypesAndGenericFilters(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if got, want := query["propertyType"], []string{"apartment", "condo"}; !slices.Equal(got, want) {
			t.Errorf("propertyType = %#v, want %#v", got, want)
		}
		if _, ok := query["propertyTypes"]; ok {
			t.Error("query must use singular propertyType, not propertyTypes")
		}
		if got := query.Get("status"); got != "active" {
			t.Errorf("status = %q, want active", got)
		}
		if got := query.Get("limit"); got != "20" {
			t.Errorf("limit = %q, want 20", got)
		}
		if got := query.Get("page"); got != "2" {
			t.Errorf("page = %q, want 2", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}

	_, err := executeCommand(t, handler, propertiesListCmd(),
		"--property-type", "apartment", "--property-type", "condo",
		"--limit", "20", "--page", "2", "--filter", "status=active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesListCmd_EncodesExactStudioFilter(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if got := query.Get("bedroomType"); got != "studio" {
			t.Errorf("bedroomType = %q, want studio", got)
		}
		if len(query["bedroomType"]) != 1 {
			t.Errorf("bedroomType values = %#v, want one value", query["bedroomType"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}

	_, err := executeCommand(t, handler, propertiesListCmd(), "--bedroom-type", "studio")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesListCmd_RejectsUnsupportedFilterValuesBeforeRequest(t *testing.T) {
	tests := map[string]struct {
		args []string
		want string
	}{
		"property type": {
			args: []string{"--property-type", "villa"},
			want: "invalid property type",
		},
		"bedroom type": {
			args: []string{"--bedroom-type", "one-bedroom"},
			want: "bedroom type must be studio",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			hit := false
			handler := func(w http.ResponseWriter, _ *http.Request) {
				hit = true
				w.WriteHeader(http.StatusOK)
			}
			_, err := executeCommand(t, handler, propertiesListCmd(), tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if hit {
				t.Fatal("invalid filter should be rejected before contacting the API")
			}
		})
	}
}

func TestPropertiesListCmd_PreservesNullablePropertyTypeOutput(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{
		"id": "p1", "title": "Legacy Studio", "address": "123 Main St",
		"propertyType": nil, "status": "active", "monthlyRent": 1500,
	}}}

	output, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/properties", http.StatusOK, data), propertiesListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Legacy Studio") {
		t.Fatalf("output missing legacy property title: %s", output)
	}
	for _, propertyType := range []string{"house", "apartment", "condo", "townhouse", "room"} {
		if strings.Contains(output, propertyType) {
			t.Errorf("nullable property type was classified as %q: %s", propertyType, output)
		}
	}
}

func TestPropertiesListCmd_OmitsEmptyGenericFilterValue(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["status"]; ok {
			t.Error("empty generic filter value must not be sent")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}

	_, err := executeCommand(t, handler, propertiesListCmd(), "--filter", "status=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesGetCmd(t *testing.T) {
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/properties/p1", http.StatusOK, map[string]any{"id": "p1"}), propertiesGetCmd(), "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesCreateCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPost, "/api/v1/properties", http.StatusCreated, map[string]any{"id": "p1"})
	_, err := executeCommand(t, handler, propertiesCreateCmd(),
		"--title", "Apt A", "--address", "123 Main St", "--monthly-rent", "1500", "--bedrooms", "2", "--bathrooms", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesCreateCmd_UsesCanonicalPayload(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		want := map[string]any{
			"title":        "Downtown Loft",
			"address":      "123 Main St",
			"monthlyRent":  float64(1800),
			"bedrooms":     float64(0),
			"bathrooms":    float64(1),
			"propertyType": "apartment",
			"isPublic":     false,
		}
		if len(body) != len(want) {
			t.Errorf("payload keys = %v, want exactly %v", body, want)
		}
		for key, value := range want {
			if body[key] != value {
				t.Errorf("body[%q] = %v, want %v", key, body[key], value)
			}
		}
		for _, legacy := range []string{"name", "type", "rent"} {
			if _, ok := body[legacy]; ok {
				t.Errorf("legacy field %q must not be sent", legacy)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "p1"}})
	}

	_, err := executeCommand(t, handler, propertiesCreateCmd(),
		"--title", "Downtown Loft",
		"--address", "123 Main St",
		"--monthly-rent", "1800",
		"--bedrooms", "0",
		"--bathrooms", "1",
		"--property-type", "apartment",
		"--is-public=false",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesCreateCmd_LegacyFlagsMapToCanonicalPayload(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		for _, legacy := range []string{"name", "type", "rent"} {
			if _, ok := body[legacy]; ok {
				t.Errorf("legacy field %q must not be sent", legacy)
			}
		}
		if body["title"] != "Legacy Name" || body["propertyType"] != "house" || body["monthlyRent"] != float64(2100) {
			t.Errorf("legacy flags were not translated canonically: %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "p1"}})
	}

	_, err := executeCommand(t, handler, propertiesCreateCmd(),
		"--name", "Legacy Name",
		"--address", "123 Main St",
		"--rent", "2100",
		"--bedrooms", "2",
		"--bathrooms", "1",
		"--type", "house",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesCreateCmd_MissingRequiredField(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	_, err := executeCommand(t, handler, propertiesCreateCmd())
	if err == nil {
		t.Error("expected error for missing required property fields")
	}
}

func TestPropertiesUpdateCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPatch, "/api/v1/properties/p1", http.StatusOK, map[string]any{"id": "p1"})
	_, err := executeCommand(t, handler, propertiesUpdateCmd(), "p1", "--name", "Apt B")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesUpdateCmd_UsesCanonicalPayload(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		want := map[string]any{
			"title":        "Updated Loft",
			"monthlyRent":  float64(1900),
			"propertyType": "condo",
		}
		if len(body) != len(want) {
			t.Errorf("payload keys = %v, want exactly %v", body, want)
		}
		for key, value := range want {
			if body[key] != value {
				t.Errorf("body[%q] = %v, want %v", key, body[key], value)
			}
		}
		for _, legacy := range []string{"name", "type", "rent"} {
			if _, ok := body[legacy]; ok {
				t.Errorf("legacy field %q must not be sent", legacy)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "p1"}})
	}

	_, err := executeCommand(t, handler, propertiesUpdateCmd(), "p1",
		"--title", "Updated Loft", "--monthly-rent", "1900", "--property-type", "condo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesListCmd_UsesCanonicalOutputFields(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{
		"id": "p1", "title": "Downtown Loft", "address": "123 Main St",
		"propertyType": "apartment", "status": "active", "monthlyRent": 1800,
	}}}
	output, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/properties", http.StatusOK, data), propertiesListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"TITLE", "PROPERTY TYPE", "MONTHLY RENT", "Downtown Loft", "apartment", "1800"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q: %s", want, output)
		}
	}
}

func TestPropertiesDeleteCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodDelete, "/api/v1/properties/p1", http.StatusNoContent, nil)
	_, err := executeCommand(t, handler, propertiesDeleteCmd(), "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Workflows ---

func TestWorkflowsListCmd(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "w1", "name": "auto-reply"}}}
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/workflows", http.StatusOK, data), workflowsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsGetCmd(t *testing.T) {
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/workflows/w1", http.StatusOK, map[string]any{"id": "w1"}), workflowsGetCmd(), "w1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsCreateCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPost, "/api/v1/workflows", http.StatusCreated, map[string]any{"id": "w1"})
	_, err := executeCommand(t, handler, workflowsCreateCmd(), "--name", "auto-reply", "--type", "notification")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsCreateCmd_MissingName(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	_, err := executeCommand(t, handler, workflowsCreateCmd())
	if err == nil {
		t.Error("expected error for missing --name")
	}
}

func TestWorkflowsUpdateCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPatch, "/api/v1/workflows/w1", http.StatusOK, map[string]any{"id": "w1"})
	_, err := executeCommand(t, handler, workflowsUpdateCmd(), "w1", "--name", "auto-reply-v2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsDeleteCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodDelete, "/api/v1/workflows/w1", http.StatusNoContent, nil)
	_, err := executeCommand(t, handler, workflowsDeleteCmd(), "w1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Conversations ---

func TestConversationsListCmd(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "cv1", "channel": "sms"}}}
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/conversations", http.StatusOK, data), conversationsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConversationsGetCmd(t *testing.T) {
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/conversations/cv1", http.StatusOK, map[string]any{"id": "cv1"}), conversationsGetCmd(), "cv1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConversationsSearchCmd(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("q"); got != "rent" {
			t.Errorf("q = %q, want rent", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}
	_, err := executeCommand(t, handler, conversationsSearchCmd(), "--query", "rent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConversationsSearchCmd_MissingQuery(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	_, err := executeCommand(t, handler, conversationsSearchCmd())
	if err == nil {
		t.Error("expected error for missing --query")
	}
}

// --- Sessions ---

func TestSessionsListCmd(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "s1", "status": "active"}}}
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/sessions", http.StatusOK, data), sessionsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionsGetCmd(t *testing.T) {
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/sessions/s1", http.StatusOK, map[string]any{"id": "s1"}), sessionsGetCmd(), "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionsReviewCmd(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/sessions/s1/review" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["rating"] != float64(5) {
			t.Errorf("rating = %v, want 5", body["rating"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}
	_, err := executeCommand(t, handler, sessionsReviewCmd(), "s1", "--rating", "5", "--notes", "great")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Showings ---

func TestShowingsListCmd(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "sh1"}}}
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/showings", http.StatusOK, data), showingsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShowingsGetCmd(t *testing.T) {
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/showings/sh1", http.StatusOK, map[string]any{"id": "sh1"}), showingsGetCmd(), "sh1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShowingsCreateCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPost, "/api/v1/showings", http.StatusCreated, map[string]any{"id": "sh1"})
	_, err := executeCommand(t, handler, showingsCreateCmd(),
		"--property-id", "p1", "--contact-id", "c1", "--scheduled-at", "2026-03-20T10:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShowingsCreateCmd_MissingRequired(t *testing.T) {
	cases := map[string][]string{
		"missing property-id":  {"--contact-id", "c1", "--scheduled-at", "2026-03-20T10:00:00Z"},
		"missing contact-id":   {"--property-id", "p1", "--scheduled-at", "2026-03-20T10:00:00Z"},
		"missing scheduled-at": {"--property-id", "p1", "--contact-id", "c1"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
			_, err := executeCommand(t, handler, showingsCreateCmd(), args...)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestShowingsUpdateCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPatch, "/api/v1/showings/sh1", http.StatusOK, map[string]any{"id": "sh1"})
	_, err := executeCommand(t, handler, showingsUpdateCmd(), "sh1", "--status", "completed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShowingsCancelCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPatch, "/api/v1/showings/sh1", http.StatusOK, map[string]any{"id": "sh1"})
	_, err := executeCommand(t, handler, showingsCancelCmd(), "sh1", "--reason", "tenant cancelled")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShowingsCheckAvailabilityCmd(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("property_id"); got != "p1" {
			t.Errorf("property_id = %q, want p1", got)
		}
		if got := r.URL.Query().Get("date"); got != "2026-03-20" {
			t.Errorf("date = %q, want 2026-03-20", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"slots": []any{"10:00", "14:00"}})
	}
	_, err := executeCommand(t, handler, showingsCheckAvailabilityCmd(), "--property-id", "p1", "--date", "2026-03-20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShowingsCheckAvailabilityCmd_MissingFlags(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	_, err := executeCommand(t, handler, showingsCheckAvailabilityCmd())
	if err == nil {
		t.Error("expected error for missing required flags")
	}
}

// --- Settings ---

func TestSettingsGetCmd(t *testing.T) {
	data := map[string]any{"followup_enabled": true, "followup_delay_hours": 24}
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/settings", http.StatusOK, data), settingsGetCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSettingsUpdateCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPatch, "/api/v1/settings", http.StatusOK, map[string]any{"status": "ok"})
	_, err := executeCommand(t, handler, settingsUpdateCmd(), "--followup-enabled", "true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSettingsUpdateCmd_NoFlags(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	_, err := executeCommand(t, handler, settingsUpdateCmd())
	if err == nil {
		t.Error("expected error when no settings specified")
	}
}

// --- Webhooks ---

func TestWebhooksListCmd(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "wh1", "url": "https://hook.test"}}}
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/webhooks", http.StatusOK, data), webhooksListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksGetCmd(t *testing.T) {
	_, err := executeCommand(t, jsonHandler(t, http.MethodGet, "/api/v1/webhooks/wh1", http.StatusOK, map[string]any{"id": "wh1"}), webhooksGetCmd(), "wh1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksCreateCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPost, "/api/v1/webhooks", http.StatusCreated, map[string]any{"id": "wh1"})
	_, err := executeCommand(t, handler, webhooksCreateCmd(), "--url", "https://hook.test", "--events", "contact.created")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksCreateCmd_MissingURL(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	_, err := executeCommand(t, handler, webhooksCreateCmd())
	if err == nil {
		t.Error("expected error for missing --url")
	}
}

func TestWebhooksUpdateCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPatch, "/api/v1/webhooks/wh1", http.StatusOK, map[string]any{"id": "wh1"})
	_, err := executeCommand(t, handler, webhooksUpdateCmd(), "wh1", "--url", "https://new.hook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksDeleteCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodDelete, "/api/v1/webhooks/wh1", http.StatusNoContent, nil)
	_, err := executeCommand(t, handler, webhooksDeleteCmd(), "wh1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksTestCmd(t *testing.T) {
	handler := jsonHandler(t, http.MethodPost, "/api/v1/webhooks/wh1/test", http.StatusOK, map[string]any{"delivered": true})
	_, err := executeCommand(t, handler, webhooksTestCmd(), "wh1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Config ---

func TestRunConfigInit(t *testing.T) {
	dir := t.TempDir()
	globalConfigFile = dir + "/config.yaml"
	err := runConfigInit(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Second call without --force should fail.
	configForce = false
	err = runConfigInit(nil, nil)
	if err == nil {
		t.Error("expected error when config exists without --force")
	}
}

func TestRunConfigSet(t *testing.T) {
	dir := t.TempDir()
	globalConfigFile = dir + "/config.yaml"
	// Init first.
	_ = runConfigInit(nil, nil)

	err := runConfigSet(nil, []string{"api_key", "ra_test123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunConfigSet_InvalidKey(t *testing.T) {
	dir := t.TempDir()
	globalConfigFile = dir + "/config.yaml"
	_ = runConfigInit(nil, nil)

	err := runConfigSet(nil, []string{"invalid_key", "value"})
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestRunConfigShow(t *testing.T) {
	dir := t.TempDir()
	globalConfigFile = dir + "/config.yaml"
	_ = runConfigInit(nil, nil)

	err := runConfigShow(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunConfigInit_WithForce(t *testing.T) {
	dir := t.TempDir()
	globalConfigFile = dir + "/config.yaml"
	_ = runConfigInit(nil, nil)

	configForce = true
	t.Cleanup(func() { configForce = false })
	err := runConfigInit(nil, nil)
	if err != nil {
		t.Fatalf("expected --force to succeed: %v", err)
	}
}

func TestRunConfigSet_BaseURL(t *testing.T) {
	dir := t.TempDir()
	globalConfigFile = dir + "/config.yaml"
	_ = runConfigInit(nil, nil)

	err := runConfigSet(nil, []string{"base_url", "https://custom.api"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunConfigShow_WithEnv(t *testing.T) {
	dir := t.TempDir()
	globalConfigFile = dir + "/config.yaml"
	_ = runConfigInit(nil, nil)

	t.Setenv("RENTALOT_API_KEY", "ra_env_key_12345")
	t.Setenv("RENTALOT_BASE_URL", "https://env.api")

	err := runConfigShow(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunConfigEdit_MissingFile(t *testing.T) {
	dir := t.TempDir()
	globalConfigFile = dir + "/nonexistent.yaml"

	// Set EDITOR to something that will fail quickly but won't block.
	t.Setenv("EDITOR", "false")
	err := runConfigEdit(nil, nil)
	// "false" command exits 1, so we expect an error.
	if err == nil {
		t.Error("expected error from editor command")
	}
}

// --- Table output path tests ---

func TestContactsListCmd_TableOutput(t *testing.T) {
	data := map[string]any{
		"data": []any{
			map[string]any{"id": "c1", "name": "Alice", "email": "alice@test.com", "phone": "555-0001", "role": "tenant"},
		},
	}
	handler := jsonHandler(t, http.MethodGet, "/api/v1/contacts", http.StatusOK, data)
	_, err := executeCommandTable(t, handler, contactsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPropertiesListCmd_TableOutput(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "p1", "name": "Apt A", "address": "123 Main"}}}
	_, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/properties", http.StatusOK, data), propertiesListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowsListCmd_TableOutput(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "w1", "name": "auto-reply"}}}
	_, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/workflows", http.StatusOK, data), workflowsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConversationsListCmd_TableOutput(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "cv1", "channel": "sms"}}}
	_, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/conversations", http.StatusOK, data), conversationsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionsListCmd_TableOutput(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "s1", "status": "active"}}}
	_, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/sessions", http.StatusOK, data), sessionsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShowingsListCmd_TableOutput(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "sh1", "status": "scheduled"}}}
	_, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/showings", http.StatusOK, data), showingsListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksListCmd_TableOutput(t *testing.T) {
	data := map[string]any{"data": []any{map[string]any{"id": "wh1", "url": "https://hook.test"}}}
	_, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/webhooks", http.StatusOK, data), webhooksListCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSettingsGetCmd_TableOutput(t *testing.T) {
	data := map[string]any{"followup_enabled": true, "followup_delay_hours": 24}
	_, err := executeCommandTable(t, jsonHandler(t, http.MethodGet, "/api/v1/settings", http.StatusOK, data), settingsGetCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConversationsSearchCmd_TableOutput(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": "cv1"}}})
	}
	_, err := executeCommandTable(t, handler, conversationsSearchCmd(), "--query", "rent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebhooksTestCmd_TableOutput(t *testing.T) {
	handler := jsonHandler(t, http.MethodPost, "/api/v1/webhooks/wh1/test", http.StatusOK, map[string]any{"delivered": true})
	_, err := executeCommandTable(t, handler, webhooksTestCmd(), "wh1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Bulk import integration ---

func TestBulkImportRun_Success(t *testing.T) {
	dir := t.TempDir()
	csvPath := dir + "/import.csv"
	if err := os.WriteFile(csvPath, []byte("address,rent\n123 Main St,1500\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/properties/bulk" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"jobId": "job-1", "status": "pending", "total": 1},
			})
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/properties/bulk/job-1" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"jobId": "job-1", "status": "completed", "total": 1, "created": 1, "failed": 0},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	cmd := &cobra.Command{Use: "bulk-import", RunE: bulkImportRun}
	cmd.Flags().String("file", csvPath, "")
	cmd.Flags().Bool("poll", true, "")

	_, err := executeCommand(t, handler, cmd, "--file", csvPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBulkImportRun_NoPoll(t *testing.T) {
	dir := t.TempDir()
	csvPath := dir + "/import.csv"
	if err := os.WriteFile(csvPath, []byte("address,rent\nAlice St,1500\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := jsonHandler(t, http.MethodPost, "/api/v1/properties/bulk", http.StatusAccepted,
		map[string]any{"data": map[string]any{"jobId": "job-1", "status": "pending", "total": 1}})

	cmd := &cobra.Command{Use: "bulk-import", RunE: bulkImportRun}
	cmd.Flags().String("file", csvPath, "")
	cmd.Flags().Bool("poll", false, "")

	_, err := executeCommand(t, handler, cmd, "--file", csvPath, "--poll=false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBulkImportRun_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	csvPath := dir + "/empty.csv"
	if err := os.WriteFile(csvPath, []byte("address,rent\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

	cmd := &cobra.Command{Use: "bulk-import", RunE: bulkImportRun}
	cmd.Flags().String("file", csvPath, "")
	cmd.Flags().Bool("poll", true, "")

	_, err := executeCommand(t, handler, cmd, "--file", csvPath)
	if err == nil {
		t.Error("expected error for empty file")
	}
}

func TestBulkImportRun_APIError(t *testing.T) {
	dir := t.TempDir()
	csvPath := dir + "/import.csv"
	if err := os.WriteFile(csvPath, []byte("address\n123 Main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	handler := jsonHandler(t, http.MethodPost, "/api/v1/properties/bulk", http.StatusBadRequest,
		map[string]any{"error": map[string]any{"code": "invalid", "message": "bad request"}})

	cmd := &cobra.Command{Use: "bulk-import", RunE: bulkImportRun}
	cmd.Flags().String("file", csvPath, "")
	cmd.Flags().Bool("poll", true, "")

	_, err := executeCommand(t, handler, cmd, "--file", csvPath)
	if err == nil {
		t.Error("expected error for API error")
	}
}

func TestPollJobStatus_Failed(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"jobId": "job-1", "status": "failed",
				"errors": []map[string]any{{"row": 1, "message": "server error"}},
			},
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(srv.Close)
	client := rentalotcli.NewClient(rentalotcli.Config{APIKey: "test", BaseURL: srv.URL})
	ctx := context.WithValue(context.Background(), clientKey, client)

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	err := pollJobStatus(cmd, "job-1")
	if err == nil {
		t.Error("expected error for failed job")
	}
	if !strings.Contains(err.Error(), "server error") {
		t.Errorf("error = %q, want to contain 'server error'", err.Error())
	}
}

func TestPollJobStatus_Completed(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"jobId": "job-1", "status": "completed", "total": 5, "created": 5, "failed": 0,
			},
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(srv.Close)
	client := rentalotcli.NewClient(rentalotcli.Config{APIKey: "test", BaseURL: srv.URL})
	ctx := context.WithValue(context.Background(), clientKey, client)

	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(ctx)

	err := pollJobStatus(cmd, "job-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- List commands with extra filters ---

func TestConversationsListCmd_WithContactID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("contact_id"); got != "c1" {
			t.Errorf("contact_id = %q, want c1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}
	_, err := executeCommand(t, handler, conversationsListCmd(), "--contact-id", "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionsListCmd_WithContactID(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("contact_id"); got != "c1" {
			t.Errorf("contact_id = %q, want c1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}
	_, err := executeCommand(t, handler, sessionsListCmd(), "--contact-id", "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShowingsListCmd_WithFilters(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("property_id"); got != "p1" {
			t.Errorf("property_id = %q, want p1", got)
		}
		if got := r.URL.Query().Get("contact_id"); got != "c1" {
			t.Errorf("contact_id = %q, want c1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}
	_, err := executeCommand(t, handler, showingsListCmd(), "--property-id", "p1", "--contact-id", "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- API error paths for all get commands ---

func TestPropertiesGetCmd_APIError(t *testing.T) {
	handler := jsonHandler(t, http.MethodGet, "/api/v1/properties/bad", http.StatusNotFound,
		map[string]any{"error": map[string]any{"code": "not_found", "message": "not found"}})
	_, err := executeCommand(t, handler, propertiesGetCmd(), "bad")
	if err == nil {
		t.Error("expected error")
	}
}

func TestWorkflowsGetCmd_APIError(t *testing.T) {
	handler := jsonHandler(t, http.MethodGet, "/api/v1/workflows/bad", http.StatusNotFound,
		map[string]any{"error": map[string]any{"code": "not_found", "message": "not found"}})
	_, err := executeCommand(t, handler, workflowsGetCmd(), "bad")
	if err == nil {
		t.Error("expected error")
	}
}

func TestSessionsGetCmd_APIError(t *testing.T) {
	handler := jsonHandler(t, http.MethodGet, "/api/v1/sessions/bad", http.StatusNotFound,
		map[string]any{"error": map[string]any{"code": "not_found", "message": "not found"}})
	_, err := executeCommand(t, handler, sessionsGetCmd(), "bad")
	if err == nil {
		t.Error("expected error")
	}
}

func TestShowingsGetCmd_APIError(t *testing.T) {
	handler := jsonHandler(t, http.MethodGet, "/api/v1/showings/bad", http.StatusNotFound,
		map[string]any{"error": map[string]any{"code": "not_found", "message": "not found"}})
	_, err := executeCommand(t, handler, showingsGetCmd(), "bad")
	if err == nil {
		t.Error("expected error")
	}
}

func TestWebhooksGetCmd_APIError(t *testing.T) {
	handler := jsonHandler(t, http.MethodGet, "/api/v1/webhooks/bad", http.StatusNotFound,
		map[string]any{"error": map[string]any{"code": "not_found", "message": "not found"}})
	_, err := executeCommand(t, handler, webhooksGetCmd(), "bad")
	if err == nil {
		t.Error("expected error")
	}
}

func TestConversationsGetCmd_APIError(t *testing.T) {
	handler := jsonHandler(t, http.MethodGet, "/api/v1/conversations/bad", http.StatusNotFound,
		map[string]any{"error": map[string]any{"code": "not_found", "message": "not found"}})
	_, err := executeCommand(t, handler, conversationsGetCmd(), "bad")
	if err == nil {
		t.Error("expected error")
	}
}

// --- List API error paths ---

func TestContactsListCmd_APIError(t *testing.T) {
	handler := jsonHandler(t, http.MethodGet, "/api/v1/contacts", http.StatusInternalServerError,
		map[string]any{"error": map[string]any{"code": "internal", "message": "server error"}})
	_, err := executeCommand(t, handler, contactsListCmd())
	if err == nil {
		t.Error("expected error")
	}
}

func TestPropertiesListCmd_APIError(t *testing.T) {
	handler := jsonHandler(t, http.MethodGet, "/api/v1/properties", http.StatusInternalServerError,
		map[string]any{"error": map[string]any{"code": "internal", "message": "server error"}})
	_, err := executeCommand(t, handler, propertiesListCmd())
	if err == nil {
		t.Error("expected error")
	}
}
