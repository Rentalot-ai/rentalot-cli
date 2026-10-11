package e2e_test

import (
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

var golden9bhUpdate = flag.Bool("update", false, "update CLI output golden files")

type golden9bhCase struct {
	args     []string
	method   string
	path     string
	fixture  string
	status   int
	exitCode int
}

func TestGolden9bhCLI(t *testing.T) {
	cases := map[string]golden9bhCase{
		"root_help":            {args: []string{"--help"}},
		"contacts_create_help": {args: []string{"contacts", "create", "--help"}},
		"properties_list_help": {args: []string{"properties", "list", "--help"}},
		"contacts_table":       {args: []string{"contacts", "list"}, method: "GET", path: "/api/v1/contacts", fixture: "contacts.json"},
		"contacts_json":        {args: []string{"contacts", "list", "--json"}, method: "GET", path: "/api/v1/contacts", fixture: "contacts.json"},
		"contacts_empty":       {args: []string{"contacts", "list"}, method: "GET", path: "/api/v1/contacts", fixture: "empty.json"},
		"properties_table":     {args: []string{"properties", "list"}, method: "GET", path: "/api/v1/properties", fixture: "properties.json"},
		"properties_json":      {args: []string{"properties", "list", "--json"}, method: "GET", path: "/api/v1/properties", fixture: "properties.json"},
		"contact_deleted":      {args: []string{"contacts", "delete", "c1"}, method: "DELETE", path: "/api/v1/contacts/c1", status: http.StatusNoContent},
		"contact_missing_arg":  {args: []string{"contacts", "get"}, exitCode: 1},
		"contact_api_error":    {args: []string{"contacts", "get", "missing"}, method: "GET", path: "/api/v1/contacts/missing", fixture: "not_found.json", status: http.StatusNotFound, exitCode: 1},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			golden9bhRun(t, name, tc)
		})
	}
}

func golden9bhRun(t *testing.T, name string, tc golden9bhCase) {
	t.Helper()
	var requests atomic.Int32
	server := golden9bhServer(t, tc, &requests)
	stdout, stderr, exitCode := golden9bhExecute(t, server.URL, tc.args)
	if exitCode != tc.exitCode {
		t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", exitCode, tc.exitCode, stdout, stderr)
	}
	wantRequests := int32(0)
	if tc.path != "" {
		wantRequests = 1
	}
	if got := requests.Load(); got != wantRequests {
		t.Fatalf("API requests = %d, want %d", got, wantRequests)
	}
	golden9bhCompare(t, name+".stdout.golden", stdout)
	golden9bhCompare(t, name+".stderr.golden", stderr)
}

func golden9bhServer(t *testing.T, tc golden9bhCase, requests *atomic.Int32) *httptest.Server {
	t.Helper()
	var body []byte
	if tc.fixture != "" {
		var err error
		body, err = os.ReadFile(filepath.Join("testdata", "golden9bh", "fixtures", tc.fixture))
		if err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != tc.method || r.URL.Path != tc.path {
			t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, tc.method, tc.path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer golden9bh-test-key" {
			t.Errorf("Authorization = %q, want fixture key", got)
		}
		status := tc.status
		if status == 0 {
			status = http.StatusOK
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	return server
}

func golden9bhExecute(t *testing.T, baseURL string, args []string) (stdout, stderr string, exitCode int) {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, "config")
	if runtime.GOOS == "darwin" {
		configDir = filepath.Join(home, "Library", "Application Support")
	}
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = home
	cmd.Env = []string{
		"HOME=" + home, "USERPROFILE=" + home,
		"XDG_CONFIG_HOME=" + configDir, "AppData=" + configDir,
		"SystemRoot=" + os.Getenv("SystemRoot"),
		"NO_COLOR=1", "TERM=dumb",
		"RENTALOT_API_KEY=golden9bh-test-key", "RENTALOT_BASE_URL=" + baseURL,
	}
	var outBuf, errBuf strings.Builder
	cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("running CLI: %v", err)
		}
		exitCode = exitErr.ExitCode()
	}
	configPath := filepath.Join(configDir, "rentalot", "config.yaml")
	normalize := strings.NewReplacer(
		strconv.Quote(configPath), strconv.Quote("<CONFIG_FILE>"),
		configPath, "<CONFIG_FILE>",
	)
	return normalize.Replace(outBuf.String()), normalize.Replace(errBuf.String()), exitCode
}

func golden9bhCompare(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden9bh", name)
	if *golden9bhUpdate {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("updating %s: %v", path, err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v (regenerate with go test ./e2e -run '^TestGolden9bhCLI$' -update)", path, err)
	}
	if got != string(want) {
		t.Errorf("output differs from %s\nwant (quoted): %s\n got (quoted): %s\nRegenerate intentional changes with go test ./e2e -run '^TestGolden9bhCLI$' -update", path, strconv.Quote(string(want)), strconv.Quote(got))
	}
}
