package e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binaryPath is set by TestMain after building the CLI binary.
var binaryPath string

func TestMain(m *testing.M) {
	// Build binary into a temp dir so subprocess tests can execute it.
	tmp, err := os.MkdirTemp("", "rentalot-e2e-*")
	if err != nil {
		panic(err)
	}
	bin := filepath.Join(tmp, "rentalot")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	out, err := exec.Command("go", "build", "-o", bin, "../cmd/rentalot/").CombinedOutput()
	if err != nil {
		panic("building binary: " + err.Error() + "\n" + string(out))
	}
	binaryPath = bin

	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// runBinary executes the CLI binary with the given args and env overrides.
// Returns stdout, stderr, and any exec error.
func runBinary(t *testing.T, env map[string]string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)

	// Inherit minimal env, then apply overrides.
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// mockAPI starts an httptest server that routes requests to handler.
func mockAPI(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// envForMock returns env vars pointing the CLI at the mock server.
func envForMock(srv *httptest.Server) map[string]string {
	return map[string]string{
		"RENTALOT_API_KEY":  "test-key",
		"RENTALOT_BASE_URL": srv.URL,
	}
}

// --- Binary-level E2E tests ---

func TestBinary_VersionFlag(t *testing.T) {
	stdout, _, err := runBinary(t, nil, "version", "--plain")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "rentalot") {
		t.Errorf("expected 'rentalot' in version output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "platform:") {
		t.Errorf("expected 'platform:' in version output, got: %s", stdout)
	}
}

func TestBinary_HelpFlag(t *testing.T) {
	stdout, _, err := runBinary(t, nil, "--help")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "rentalot") {
		t.Errorf("expected 'rentalot' in help output, got: %s", stdout)
	}
	// Should list subcommands.
	for _, sub := range []string{"contacts", "properties", "workflows", "config", "version"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("expected %q in help output", sub)
		}
	}
}

func TestBinary_CanonicalWriteHelp(t *testing.T) {
	propertyHelp, _, err := runBinary(t, nil, "properties", "create", "--help")
	if err != nil {
		t.Fatalf("properties help exit error: %v", err)
	}
	for _, want := range []string{"--address", "--monthly-rent", "--bedrooms", "--bathrooms", "--property-type"} {
		if !strings.Contains(propertyHelp, want) {
			t.Errorf("property help missing %q: %s", want, propertyHelp)
		}
	}
	for _, want := range []string{"Trial writes must remain private", "omit --property-type"} {
		if !strings.Contains(propertyHelp, want) {
			t.Errorf("property help missing trial guidance %q: %s", want, propertyHelp)
		}
	}
	for _, legacy := range []string{"--name", "--type", "--rent"} {
		if strings.Contains(propertyHelp, legacy) {
			t.Errorf("deprecated property flag %q should stay hidden from help: %s", legacy, propertyHelp)
		}
	}

	contactHelp, _, err := runBinary(t, nil, "contacts", "create", "--help")
	if err != nil {
		t.Fatalf("contacts help exit error: %v", err)
	}
	for _, want := range []string{"--name", "--email", "--phone", "--role"} {
		if !strings.Contains(contactHelp, want) {
			t.Errorf("contact help missing %q: %s", want, contactHelp)
		}
	}
	for _, want := range []string{"at least one of --email or --phone"} {
		if !strings.Contains(contactHelp, want) {
			t.Errorf("contact help missing create guidance %q: %s", want, contactHelp)
		}
	}
	if strings.Contains(contactHelp, "--type") {
		t.Errorf("deprecated contact flag --type should stay hidden from help: %s", contactHelp)
	}
}

func TestBinary_UnknownCommand(t *testing.T) {
	_, stderr, err := runBinary(t, nil, "nonexistent")
	if err == nil {
		t.Fatal("expected non-zero exit for unknown command")
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Errorf("expected 'unknown command' in stderr, got: %s", stderr)
	}
}

func TestBinary_ContactsList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("auth header = %q, want Bearer test-key", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{"id": "c1", "name": "Alice", "email": "alice@test.com", "phone": "555-0001", "role": "tenant"},
				map[string]any{"id": "c2", "name": "Bob", "email": "bob@test.com", "phone": "555-0002", "role": "landlord"},
			},
		})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "contacts", "list", "--json")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "Alice") || !strings.Contains(stdout, "Bob") {
		t.Errorf("expected contact names in output, got: %s", stdout)
	}
}

func TestBinary_ContactsList_TableOutput(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{"id": "c1", "name": "Alice", "email": "alice@test.com", "phone": "555-0001", "role": "tenant"},
			},
		})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "contacts", "list")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	// Table output should have headers.
	if !strings.Contains(stdout, "ID") || !strings.Contains(stdout, "NAME") {
		t.Errorf("expected table headers in output, got: %s", stdout)
	}
	for _, want := range []string{"Alice", "ROLE", "tenant"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in table output, got: %s", want, stdout)
		}
	}
}

func TestBinary_ContactsGet(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts/c1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "c1", "name": "Alice"},
		})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "contacts", "get", "c1")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "Alice") {
		t.Errorf("expected 'Alice' in output, got: %s", stdout)
	}
}

func TestBinary_ContactsCreate(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "NewContact" {
			t.Errorf("name = %v, want NewContact", body["name"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": "c99", "name": "NewContact"},
		})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "contacts", "create", "--name", "NewContact", "--email", "new@test.com")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "c99") {
		t.Errorf("expected created contact id in output, got: %s", stdout)
	}
}

func TestBinary_ContactsCreate_CanonicalPayload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts", func(w http.ResponseWriter, r *http.Request) {
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
			t.Errorf("payload = %v, want exactly %v", body, want)
		}
		for key, value := range want {
			if body[key] != value {
				t.Errorf("body[%q] = %v, want %v", key, body[key], value)
			}
		}
		if _, ok := body["type"]; ok {
			t.Error("legacy contact field type must not be sent")
		}
		if _, ok := body["phone"]; ok {
			t.Error("empty phone must be omitted")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "c100"}})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "contacts", "create",
		"--name", "Ada Lovelace", "--email", "ada@example.com", "--role", "tenant", "--status", "prospect")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "c100") {
		t.Errorf("expected created contact id in output, got: %s", stdout)
	}
}

func TestBinary_ContactsCreate_MissingName(t *testing.T) {
	srv := mockAPI(t, http.NewServeMux()) // won't be hit
	_, stderr, err := runBinary(t, envForMock(srv), "contacts", "create")
	if err == nil {
		t.Fatal("expected non-zero exit for missing --name")
	}
	if !strings.Contains(stderr, "name") {
		t.Errorf("expected 'name' error in stderr, got: %s", stderr)
	}
}

func TestBinary_ContactsDelete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts/c1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "contacts", "delete", "c1")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(strings.ToLower(stdout), "deleted") {
		t.Errorf("expected 'deleted' confirmation, got: %s", stdout)
	}
}

func TestBinary_PropertiesList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/properties", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{"id": "p1", "title": "Oak Apartments", "address": "123 Oak St", "propertyType": "apartment", "status": "active", "monthlyRent": 1500},
			},
		})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "properties", "list", "--json")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "Oak Apartments") {
		t.Errorf("expected property name in output, got: %s", stdout)
	}
}

func TestBinary_PropertiesList_TableOutput(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/properties", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{map[string]any{
				"id": "p1", "title": "Oak Apartments", "address": "123 Oak St",
				"propertyType": "apartment", "status": "active", "monthlyRent": 1500,
			}},
		})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "properties", "list")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	for _, want := range []string{"TITLE", "PROPERTY TYPE", "MONTHLY RENT", "Oak Apartments", "apartment", "1500"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected %q in table output, got: %s", want, stdout)
		}
	}
}

func TestBinary_PropertiesCreate_CanonicalPayload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/properties", func(w http.ResponseWriter, r *http.Request) {
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
			t.Errorf("payload = %v, want exactly %v", body, want)
		}
		for key, value := range want {
			if body[key] != value {
				t.Errorf("body[%q] = %v, want %v", key, body[key], value)
			}
		}
		for _, legacy := range []string{"name", "type", "rent"} {
			if _, ok := body[legacy]; ok {
				t.Errorf("legacy property field %q must not be sent", legacy)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "p100"}})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "properties", "create",
		"--title", "Downtown Loft", "--address", "123 Main St", "--monthly-rent", "1800",
		"--bedrooms", "0", "--bathrooms", "1", "--property-type", "apartment", "--is-public=false")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "p100") {
		t.Errorf("expected created property id in output, got: %s", stdout)
	}
}

func TestBinary_PropertiesCreate_TrialSafePayload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/properties", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		want := map[string]any{
			"title":       "Trial Loft",
			"address":     "123 Trial St",
			"monthlyRent": float64(1600),
			"bedrooms":    float64(0),
			"bathrooms":   float64(1),
			"isPublic":    false,
		}
		if len(body) != len(want) {
			t.Errorf("trial payload = %v, want exactly %v", body, want)
		}
		for key, value := range want {
			if body[key] != value {
				t.Errorf("body[%q] = %v, want %v", key, body[key], value)
			}
		}
		for _, forbidden := range []string{"name", "type", "rent", "propertyType", "imageUrls"} {
			if _, ok := body[forbidden]; ok {
				t.Errorf("trial property field %q must not be sent", forbidden)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "ptrial"}})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "properties", "create",
		"--title", "Trial Loft", "--address", "123 Trial St", "--monthly-rent", "1600",
		"--bedrooms", "0", "--bathrooms", "1", "--is-public=false")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "ptrial") {
		t.Errorf("expected trial property id in output, got: %s", stdout)
	}
}

func TestBinary_PropertiesUpdate_CanonicalPayload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/properties/p100", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
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
			t.Errorf("payload = %v, want exactly %v", body, want)
		}
		for key, value := range want {
			if body[key] != value {
				t.Errorf("body[%q] = %v, want %v", key, body[key], value)
			}
		}
		for _, legacy := range []string{"name", "type", "rent"} {
			if _, ok := body[legacy]; ok {
				t.Errorf("legacy property field %q must not be sent", legacy)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "p100"}})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "properties", "update", "p100",
		"--title", "Updated Loft", "--monthly-rent", "1900", "--property-type", "condo")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "p100") {
		t.Errorf("expected updated property id in output, got: %s", stdout)
	}
}

func TestBinary_ContactsUpdate_CanonicalPayload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts/c100", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body["role"] != "landlord" || len(body) != 1 {
			t.Errorf("payload = %v, want only canonical role", body)
		}
		if _, ok := body["type"]; ok {
			t.Error("legacy contact field type must not be sent")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "c100"}})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "contacts", "update", "c100", "--role", "landlord")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "c100") {
		t.Errorf("expected updated contact id in output, got: %s", stdout)
	}
}

func TestBinary_429TimedRetryGuidance(t *testing.T) {
	requests := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts", func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/problem+json")
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "https://rentalot.ai/problems/rate-limited",
			"title":  "Rate Limited",
			"status": 429,
			"detail": "Write rate limit exceeded for contacts (5/min)",
		})
	})

	srv := mockAPI(t, mux)
	_, stderr, err := runBinary(t, envForMock(srv), "contacts", "list")
	if err == nil {
		t.Fatal("expected non-zero exit for 429")
	}
	if requests != 1 {
		t.Errorf("requests = %d, want no automatic retry", requests)
	}
	if !strings.Contains(stderr, "retry after 3 seconds") {
		t.Errorf("stderr = %q, want timed retry guidance", stderr)
	}
}

func TestBinary_429TerminalTrialGuidance(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "https://rentalot.ai/problems/rate-limited",
			"title":  "Rate Limited",
			"status": 429,
			"detail": "Trial lifetime quota exhausted; upgrade to continue",
		})
	})

	srv := mockAPI(t, mux)
	_, stderr, err := runBinary(t, envForMock(srv), "contacts", "list")
	if err == nil {
		t.Fatal("expected non-zero exit for terminal trial quota")
	}
	if !strings.Contains(stderr, "key rotation does not reset") {
		t.Errorf("stderr = %q, want account-wide quota guidance", stderr)
	}
}

func TestBinary_401ExpiryRevocationGuidance(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts/c1", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":   "https://rentalot.ai/problems/unauthorized",
			"title":  "Unauthorized",
			"status": 401,
			"detail": "Missing or invalid API key",
		})
	})

	srv := mockAPI(t, mux)
	_, stderr, err := runBinary(t, envForMock(srv), "contacts", "get", "c1")
	if err == nil {
		t.Fatal("expected non-zero exit for unauthorized key")
	}
	if !strings.Contains(stderr, "expired or revoked") {
		t.Errorf("stderr = %q, want expiry/revocation guidance", stderr)
	}
}

func TestBinary_PropertiesCreate_MissingRequiredField(t *testing.T) {
	srv := mockAPI(t, http.NewServeMux())
	_, stderr, err := runBinary(t, envForMock(srv), "properties", "create")
	if err == nil {
		t.Fatal("expected non-zero exit for missing required property fields")
	}
	if !strings.Contains(stderr, "address") {
		t.Errorf("expected address error in stderr, got: %s", stderr)
	}
}

func TestBinary_APIError_Propagation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts/missing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": "not_found", "message": "contact not found"},
		})
	})

	srv := mockAPI(t, mux)
	_, stderr, err := runBinary(t, envForMock(srv), "contacts", "get", "missing")
	if err == nil {
		t.Fatal("expected non-zero exit for 404")
	}
	if !strings.Contains(stderr, "not_found") && !strings.Contains(stderr, "contact not found") {
		t.Errorf("expected API error in stderr, got: %s", stderr)
	}
}

func TestBinary_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/contacts", func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		page := r.URL.Query().Get("page")
		if limit != "5" {
			t.Errorf("limit = %q, want 5", limit)
		}
		if page != "2" {
			t.Errorf("page = %q, want 2", page)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	})

	srv := mockAPI(t, mux)
	_, _, err := runBinary(t, envForMock(srv), "contacts", "list", "--limit", "5", "--page", "2", "--json")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
}

func TestBinary_WorkflowsList(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/workflows", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{"id": "w1", "name": "Onboarding", "status": "active"},
			},
		})
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "workflows", "list", "--json")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "Onboarding") {
		t.Errorf("expected workflow name in output, got: %s", stdout)
	}
}

func TestBinary_ConfigShow(t *testing.T) {
	// Point at a nonexistent config file so it falls back to env.
	env := map[string]string{
		"RENTALOT_API_KEY":  "sk-test-123",
		"RENTALOT_BASE_URL": "https://mock.rentalot.ai",
	}
	stdout, _, err := runBinary(t, env, "config", "show", "--config", "/tmp/nonexistent-rentalot-cfg.yaml")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(stdout, "sk-test") || !strings.Contains(stdout, "mock.rentalot.ai") {
		t.Errorf("expected config values in output, got: %s", stdout)
	}
}

func TestBinary_NoColorFlag(t *testing.T) {
	stdout, _, err := runBinary(t, nil, "--no-color", "version", "--plain")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	// With --no-color and --plain, output should have no ANSI escape codes.
	if strings.Contains(stdout, "\033[") {
		t.Errorf("expected no ANSI codes with --no-color, got: %s", stdout)
	}
}

func TestBinary_ContactsGet_MissingArg(t *testing.T) {
	srv := mockAPI(t, http.NewServeMux())
	_, stderr, err := runBinary(t, envForMock(srv), "contacts", "get")
	if err == nil {
		t.Fatal("expected non-zero exit for missing ID argument")
	}
	if !strings.Contains(stderr, "accepts 1 arg") {
		t.Errorf("expected arg count error in stderr, got: %s", stderr)
	}
}

func TestBinary_PropertyDelete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/properties/p1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := mockAPI(t, mux)
	stdout, _, err := runBinary(t, envForMock(srv), "properties", "delete", "p1")
	if err != nil {
		t.Fatalf("exit error: %v", err)
	}
	if !strings.Contains(strings.ToLower(stdout), "deleted") {
		t.Errorf("expected 'deleted' confirmation, got: %s", stdout)
	}
}

func TestBinary_RFC9457_ErrorFormat(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/properties/bad", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{
			"type":   "validation_error",
			"detail": "address is required",
		})
	})

	srv := mockAPI(t, mux)
	_, stderr, err := runBinary(t, envForMock(srv), "properties", "get", "bad")
	if err == nil {
		t.Fatal("expected non-zero exit for 422")
	}
	if !strings.Contains(stderr, "address is required") {
		t.Errorf("expected RFC 9457 detail in stderr, got: %s", stderr)
	}
}
