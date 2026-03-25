package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
	"github.com/joho/godotenv"
)

// env is the resolved E2E environment (dev or prod).
type env struct {
	BaseURL string
	APIKey  string
	Name    string // "dev" or "prod"
}

func loadEnv(t *testing.T) env {
	t.Helper()

	// Load .env from project root (best-effort).
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(thisFile))
	_ = godotenv.Load(filepath.Join(root, ".env"))

	target := os.Getenv("RENTALOT_E2E_TARGET")
	if target == "" {
		target = "dev"
	}

	switch target {
	case "dev":
		return env{
			BaseURL: requireEnv(t, "RENTALOT_E2E_DEV_BASE_URL"),
			APIKey:  requireEnv(t, "RENTALOT_E2E_DEV_API_KEY"),
			Name:    "dev",
		}
	case "prod":
		return env{
			BaseURL: requireEnv(t, "RENTALOT_E2E_PROD_BASE_URL"),
			APIKey:  requireEnv(t, "RENTALOT_E2E_PROD_API_KEY"),
			Name:    "prod",
		}
	default:
		t.Fatalf("invalid RENTALOT_E2E_TARGET=%q (want dev or prod)", target)
		return env{}
	}
}

func requireEnv(t *testing.T, key string) string {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		t.Skipf("%s not set — skipping e2e tests (see .env.example)", key)
	}
	return v
}

func newE2EClient(t *testing.T) *rentalotcli.Client {
	t.Helper()
	e := loadEnv(t)
	t.Logf("E2E target: %s (%s)", e.Name, e.BaseURL)
	return rentalotcli.NewClient(rentalotcli.Config{
		APIKey:  e.APIKey,
		BaseURL: e.BaseURL,
	})
}

// decodeJSON reads JSON from a response into dst and fails the test on error.
func decodeJSON(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
}

// assertStatus checks the HTTP status code.
func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		t.Errorf("status = %d, want %d", resp.StatusCode, want)
	}
}

// --- Tests ---

func TestE2E_PropertiesList(t *testing.T) {
	client := newE2EClient(t)
	ctx := context.Background()

	resp, err := client.Get(ctx, "/api/v1/properties", nil)
	if err != nil {
		t.Fatalf("listing properties: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	decodeJSON(t, resp, &result)

	// Seeded data has 2 properties — just verify we get a non-empty list.
	data, ok := result["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", result["data"])
	}
	if len(data) == 0 {
		t.Error("expected at least 1 property, got 0")
	}
	t.Logf("got %d properties", len(data))
}

func TestE2E_ContactsList(t *testing.T) {
	client := newE2EClient(t)
	ctx := context.Background()

	resp, err := client.Get(ctx, "/api/v1/contacts", nil)
	if err != nil {
		t.Fatalf("listing contacts: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	decodeJSON(t, resp, &result)

	data, ok := result["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", result["data"])
	}
	if len(data) == 0 {
		t.Error("expected at least 1 contact, got 0")
	}
	t.Logf("got %d contacts", len(data))
}

func TestE2E_SettingsGet(t *testing.T) {
	client := newE2EClient(t)
	ctx := context.Background()

	resp, err := client.Get(ctx, "/api/v1/settings", nil)
	if err != nil {
		t.Fatalf("getting settings: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	decodeJSON(t, resp, &result)

	// Settings should have a data object.
	if _, ok := result["data"]; !ok {
		t.Error("expected 'data' key in settings response")
	}
}

func TestE2E_ConversationsList(t *testing.T) {
	client := newE2EClient(t)
	ctx := context.Background()

	resp, err := client.Get(ctx, "/api/v1/conversations", nil)
	if err != nil {
		t.Fatalf("listing conversations: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	decodeJSON(t, resp, &result)

	data, ok := result["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", result["data"])
	}
	if len(data) == 0 {
		t.Error("expected at least 1 conversation, got 0")
	}
	t.Logf("got %d conversations", len(data))
}

func TestE2E_WorkflowsList(t *testing.T) {
	client := newE2EClient(t)
	ctx := context.Background()

	resp, err := client.Get(ctx, "/api/v1/workflows", nil)
	if err != nil {
		t.Fatalf("listing workflows: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var result map[string]any
	decodeJSON(t, resp, &result)

	data, ok := result["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", result["data"])
	}
	if len(data) == 0 {
		t.Error("expected at least 1 workflow, got 0")
	}
	t.Logf("got %d workflows", len(data))
}

func TestE2E_Unauthorized(t *testing.T) {
	e := loadEnv(t)
	badClient := rentalotcli.NewClient(rentalotcli.Config{
		APIKey:  "ra_invalid_key_that_should_not_work",
		BaseURL: e.BaseURL,
	})

	resp, err := badClient.Get(context.Background(), "/api/v1/properties", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d (bad API key should be rejected)", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestE2E_PropertyCRUD(t *testing.T) {
	client := newE2EClient(t)
	ctx := context.Background()

	// Create
	newProp := map[string]any{
		"address":     "999 E2E Test Lane",
		"city":        "Testville",
		"state":       "TX",
		"zip":         "99999",
		"monthlyRent": 1234,
		"bedrooms":    1,
		"bathrooms":   1,
		"status":      "inactive",
	}
	resp, err := client.Post(ctx, "/api/v1/properties", newProp)
	if err != nil {
		t.Fatalf("creating property: %v", err)
	}
	assertStatus(t, resp, http.StatusCreated)

	var created map[string]any
	decodeJSON(t, resp, &created)
	data, _ := created["data"].(map[string]any)
	id, _ := data["id"].(string)
	if id == "" {
		t.Fatal("created property has no id")
	}
	t.Logf("created property %s", id)

	// Get
	resp, err = client.Get(ctx, fmt.Sprintf("/api/v1/properties/%s", id), nil)
	if err != nil {
		t.Fatalf("getting property: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)

	var got map[string]any
	decodeJSON(t, resp, &got)
	gotData, _ := got["data"].(map[string]any)
	if gotData["address"] != "999 E2E Test Lane" {
		t.Errorf("address = %v, want %q", gotData["address"], "999 E2E Test Lane")
	}

	// Update
	resp, err = client.Patch(ctx, fmt.Sprintf("/api/v1/properties/%s", id), map[string]any{
		"address": "999 E2E Test Lane (Updated)",
	})
	if err != nil {
		t.Fatalf("updating property: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Delete (clean up)
	resp, err = client.Delete(ctx, fmt.Sprintf("/api/v1/properties/%s", id))
	if err != nil {
		t.Fatalf("deleting property: %v", err)
	}
	// Accept 200 or 204.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete status = %d, want 200 or 204", resp.StatusCode)
	}
	_ = resp.Body.Close()
	t.Log("property CRUD cycle complete")
}

func TestE2E_ContactCRUD(t *testing.T) {
	client := newE2EClient(t)
	ctx := context.Background()

	// Create
	newContact := map[string]any{
		"name":   "E2E Test Contact",
		"email":  "e2e-test@example.com",
		"phone":  "+15559999999",
		"status": "prospect",
		"source": "web",
	}
	resp, err := client.Post(ctx, "/api/v1/contacts", newContact)
	if err != nil {
		t.Fatalf("creating contact: %v", err)
	}
	assertStatus(t, resp, http.StatusCreated)

	var created map[string]any
	decodeJSON(t, resp, &created)
	data, _ := created["data"].(map[string]any)
	id, _ := data["id"].(string)
	if id == "" {
		t.Fatal("created contact has no id")
	}
	t.Logf("created contact %s", id)

	// Get
	resp, err = client.Get(ctx, fmt.Sprintf("/api/v1/contacts/%s", id), nil)
	if err != nil {
		t.Fatalf("getting contact: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Update
	resp, err = client.Patch(ctx, fmt.Sprintf("/api/v1/contacts/%s", id), map[string]any{
		"name": "E2E Test Contact (Updated)",
	})
	if err != nil {
		t.Fatalf("updating contact: %v", err)
	}
	assertStatus(t, resp, http.StatusOK)
	_ = resp.Body.Close()

	// Delete (clean up)
	resp, err = client.Delete(ctx, fmt.Sprintf("/api/v1/contacts/%s", id))
	if err != nil {
		t.Fatalf("deleting contact: %v", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete status = %d, want 200 or 204", resp.StatusCode)
	}
	_ = resp.Body.Close()
	t.Log("contact CRUD cycle complete")
}
