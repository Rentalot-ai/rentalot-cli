package e2e_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBinaryImage33mHelp(t *testing.T) {
	for _, operation := range []string{"presign", "confirm", "presign-batch", "confirm-batch"} {
		stdout, stderr, err := runBinary(t, nil, "properties", "images", operation, "--help")
		if err != nil {
			t.Fatalf("%s help: %v: %s", operation, err, stderr)
		}
		for _, text := range []string{"image/jpeg", "image/png", "image/webp", "HEIC is unsupported", "NSFW", "HTTP 422", "deletes"} {
			if !strings.Contains(stdout, text) {
				t.Errorf("%s help missing %q: %s", operation, text, stdout)
			}
		}
	}
}

func TestBinaryImage33mConfirm422(t *testing.T) {
	for _, operation := range []string{"confirm", "confirm-batch"} {
		t.Run(operation, func(t *testing.T) {
			calls := 0
			srv := mockAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/properties/p1/images/"+operation {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"type":"validation_error","detail":"Image rejected: NSFW content detected"}`))
			}))
			args := []string{"properties", "images", operation, "p1"}
			if operation == "confirm-batch" {
				path := filepath.Join(t.TempDir(), "confirm.json")
				if err := os.WriteFile(path, []byte(`{"images":[{"r2Key":"properties/p1/photo","contentType":"image/jpeg","sizeBytes":256}]}`), 0o600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--file", path)
			} else {
				args = append(args, "--r2-key", "properties/p1/photo", "--content-type", "image/jpeg", "--size-bytes", "256")
			}
			stdout, stderr, err := runBinary(t, envForMock(srv), args...)
			if err == nil {
				t.Fatal("422 returned a successful exit")
			}
			for _, text := range []string{"HTTP 422", "NSFW content detected", "rejected upload was deleted", "presign/upload it again"} {
				if !strings.Contains(stderr, text) {
					t.Errorf("stderr missing %q: %s", text, stderr)
				}
			}
			if calls != 1 || strings.Contains(stdout, `"data"`) {
				t.Errorf("calls = %d, stdout = %q", calls, stdout)
			}
		})
	}
}

func TestBinaryImage33mHEICRejectedLocally(t *testing.T) {
	srv := mockAPI(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("HEIC request reached API")
	}))
	_, stderr, err := runBinary(t, envForMock(srv), "properties", "images", "presign", "p1", "--file-name", "photo.heic", "--content-type", "image/heic", "--size-bytes", "256")
	if err == nil || !strings.Contains(stderr, "convert HEIC before uploading") {
		t.Fatalf("exit = %v, stderr = %s", err, stderr)
	}
}
