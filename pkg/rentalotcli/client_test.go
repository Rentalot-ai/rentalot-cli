package rentalotcli_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*rentalotcli.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := rentalotcli.Config{APIKey: "test-key", BaseURL: srv.URL}
	return rentalotcli.NewClient(cfg), srv
}

func TestClient_TimeoutOverrideBoundsStalledRequests(t *testing.T) {
	requests := map[string]func(*rentalotcli.Client, context.Context) (*http.Response, error){
		"GET": func(c *rentalotcli.Client, ctx context.Context) (*http.Response, error) {
			return c.Get(ctx, "/stalled", nil)
		},
		"POST": func(c *rentalotcli.Client, ctx context.Context) (*http.Response, error) {
			return c.Post(ctx, "/stalled", map[string]string{"name": "test"})
		},
		"PATCH": func(c *rentalotcli.Client, ctx context.Context) (*http.Response, error) {
			return c.Patch(ctx, "/stalled", map[string]string{"name": "test"})
		},
		"DELETE": func(c *rentalotcli.Client, ctx context.Context) (*http.Response, error) {
			return c.Delete(ctx, "/stalled")
		},
	}
	for method, request := range requests {
		t.Run(method, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				<-r.Context().Done()
			}))
			defer srv.Close()
			client := rentalotcli.NewClient(rentalotcli.Config{BaseURL: srv.URL, Timeout: 50 * time.Millisecond})
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			resp, err := request(client, ctx)
			if resp != nil {
				_ = resp.Body.Close()
			}
			if ctx.Err() != nil {
				t.Fatal("request reached the caller backstop instead of the configured timeout")
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("error = %v, want deadline exceeded", err)
			}
		})
	}
}

func TestClient_TimeoutBoundsStalledResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()
	client := rentalotcli.NewClient(rentalotcli.Config{BaseURL: srv.URL, Timeout: 100 * time.Millisecond})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := client.Get(ctx, "/stalled-body", nil)
	if err != nil {
		t.Fatalf("receiving headers: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, err = io.ReadAll(resp.Body)
	if ctx.Err() != nil {
		t.Fatal("body read reached the caller backstop instead of the configured timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("body read error = %v, want deadline exceeded", err)
	}
}

func TestClient_PreservesEarlierContextDeadline(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	resp, err := client.Get(ctx, "/stalled", nil)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("request error = %v, context error = %v; want caller deadline exceeded", err, ctx.Err())
	}
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
	_ = resp.Body.Close()
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
	_ = resp.Body.Close()
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
	_ = resp.Body.Close()
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

func TestDecodeError_PreservesStatusAndRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     http.Header{"Retry-After": []string{"3"}},
		Body: io.NopCloser(strings.NewReader(`{
			"type":"https://rentalot.ai/problems/rate-limited",
			"title":"Rate Limited",
			"status":429,
			"detail":"Write rate limit exceeded for contacts (5/min)"
		}`)),
	}

	err := rentalotcli.DecodeError(resp)
	apiErr, ok := err.(*rentalotcli.APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	value := reflect.ValueOf(apiErr).Elem()
	status := value.FieldByName("StatusCode")
	if !status.IsValid() || status.Int() != http.StatusTooManyRequests {
		t.Fatalf("StatusCode = %v, want %d", status, http.StatusTooManyRequests)
	}
	retryAfter := value.FieldByName("RetryAfter")
	if !retryAfter.IsValid() || retryAfter.Int() != 3 {
		t.Fatalf("RetryAfter = %v, want 3", retryAfter)
	}
	if !apiErr.Retryable() {
		t.Error("Retryable() = false, want true for a timed 429")
	}
	if !strings.Contains(err.Error(), "retry after 3 seconds") {
		t.Errorf("error = %q, want timed retry guidance", err)
	}
}

func TestDecodeError_AuthenticationRecoveryGuidance(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(strings.NewReader(`{"type":"https://rentalot.ai/problems/unauthorized","detail":"Missing or invalid API key"}`)),
	}

	err := rentalotcli.DecodeError(resp)
	if !strings.Contains(err.Error(), "expired or revoked") {
		t.Errorf("error = %q, want expiry/revocation guidance", err)
	}
}

func TestDecodeError_TerminalQuotaRecoveryGuidance(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Body: io.NopCloser(strings.NewReader(`{
			"type":"https://rentalot.ai/problems/rate-limited",
			"title":"Rate Limited",
			"status":429,
			"detail":"Trial lifetime quota exhausted; upgrade to continue"
		}`)),
	}

	err := rentalotcli.DecodeError(resp)
	apiErr, ok := err.(*rentalotcli.APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Retryable() {
		t.Error("Retryable() = true, want false without a timed Retry-After")
	}
	if !strings.Contains(err.Error(), "key rotation does not reset") {
		t.Errorf("error = %q, want account-wide quota recovery guidance", err)
	}
}
