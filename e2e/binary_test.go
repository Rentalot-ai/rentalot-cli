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
	defer os.RemoveAll(tmp)

	bin := filepath.Join(tmp, "rentalot")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	out, err := exec.Command("go", "build", "-o", bin, "../cmd/rentalot/").CombinedOutput()
	if err != nil {
		panic("building binary: " + err.Error() + "\n" + string(out))
	}
	binaryPath = bin

	os.Exit(m.Run())
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
				map[string]any{"id": "c1", "name": "Alice", "email": "alice@test.com", "phone": "555-0001", "type": "tenant"},
				map[string]any{"id": "c2", "name": "Bob", "email": "bob@test.com", "phone": "555-0002", "type": "owner"},
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
				map[string]any{"id": "c1", "name": "Alice", "email": "alice@test.com", "phone": "555-0001", "type": "tenant"},
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
	if !strings.Contains(stdout, "Alice") {
		t.Errorf("expected 'Alice' in table output, got: %s", stdout)
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
				map[string]any{"id": "p1", "name": "Oak Apartments", "address": "123 Oak St", "type": "apartment", "status": "available", "rent": "1500"},
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

func TestBinary_PropertiesCreate_MissingName(t *testing.T) {
	srv := mockAPI(t, http.NewServeMux())
	_, stderr, err := runBinary(t, envForMock(srv), "properties", "create")
	if err == nil {
		t.Fatal("expected non-zero exit for missing --name")
	}
	if !strings.Contains(stderr, "name") {
		t.Errorf("expected 'name' error in stderr, got: %s", stderr)
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
