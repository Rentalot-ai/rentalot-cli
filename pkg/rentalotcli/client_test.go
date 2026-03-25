package rentalotcli_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*rentalotcli.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := rentalotcli.Config{APIKey: "test-key", BaseURL: srv.URL}
	return rentalotcli.NewClient(cfg), srv
}

func TestGet_InjectsAuthHeader(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if got != "Bearer test-key" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-key")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	resp, err := client.Get(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestGet_WithQueryParams(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Errorf("limit = %q, want %q", got, "10")
		}
		if got := r.URL.Query().Get("status"); got != "active" {
			t.Errorf("status = %q, want %q", got, "active")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	params := rentalotcli.QueryParams{"limit": "10", "status": "active"}
	resp, err := client.Get(context.Background(), "/test", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestGet_EmptyParamsIgnored(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("expected no query string, got %q", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	params := rentalotcli.QueryParams{"empty": ""}
	resp, err := client.Get(context.Background(), "/test", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestPost_SendsJSONBody(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
		}
		if body["name"] != "test" {
			t.Errorf("name = %v, want %q", body["name"], "test")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"123"}`))
	})

	resp, err := client.Post(context.Background(), "/test", map[string]any{"name": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestPatch_SendsJSONBody(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"status":"active"}`+"\n" {
			t.Errorf("body = %q", body)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})

	resp, err := client.Patch(context.Background(), "/test/1", map[string]any{"status": "active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestDelete_SendsDeleteRequest(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q, want DELETE", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	resp, err := client.Delete(context.Background(), "/test/1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestDecodeError_EnvelopeShape(t *testing.T) {
	cases := map[string]struct {
		body    string
		wantMsg string
	}{
		"error envelope": {
			body:    `{"error":{"code":"not_found","message":"resource not found"}}`,
			wantMsg: "not_found: resource not found",
		},
		"rfc9457": {
			body:    `{"type":"urn:problem:validation","detail":"invalid email"}`,
			wantMsg: "urn:problem:validation: invalid email",
		},
		"unknown": {
			body:    `{"something":"else"}`,
			wantMsg: "unknown: HTTP 400",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(tc.body)),
			}
			err := rentalotcli.DecodeError(resp)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tc.wantMsg {
				t.Errorf("error = %q, want %q", err.Error(), tc.wantMsg)
			}
		})
	}
}
