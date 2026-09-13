package rentalotcli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// APIError represents an error returned by the Rentalot API.
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	// RetryAfter is the server-provided number of seconds before a retry.
	// A zero value means that no timed retry was provided.
	RetryAfter int `json:"retry_after,omitempty"`
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("%s: %s", e.Code, e.Message)
	switch {
	case e.StatusCode == http.StatusUnauthorized:
		return message + " (check that the API key is valid and not expired or revoked)"
	case e.StatusCode == http.StatusTooManyRequests && e.RetryAfter > 0:
		return fmt.Sprintf("%s (retry after %d seconds; do not retry automatically)", message, e.RetryAfter)
	case e.StatusCode == http.StatusTooManyRequests && isTrialQuotaMessage(e.Message):
		return message + " (no timed retry; key rotation does not reset account-wide trial quota)"
	case e.StatusCode == http.StatusTooManyRequests:
		return message + " (no timed retry was provided; check the account quota before retrying)"
	default:
		return message
	}
}

// Retryable reports whether the server supplied a safe, timed retry interval.
// Callers must still avoid automatic retries for mutations unless they have an
// idempotency contract appropriate for their operation.
func (e *APIError) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests && e.RetryAfter > 0
}

func isTrialQuotaMessage(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "trial") || strings.Contains(message, "lifetime")
}

// Client is a thin HTTP wrapper for the Rentalot REST API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client from the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{},
	}
}

// QueryParams maps query parameter names to their string values.
type QueryParams map[string]string

// Get performs an authenticated GET request to the given path.
func (c *Client) Get(ctx context.Context, path string, params QueryParams) (*http.Response, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			if v != "" {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	return c.do(ctx, http.MethodGet, u.String(), nil)
}

// Post performs an authenticated POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.doJSON(ctx, http.MethodPost, c.baseURL+path, body)
}

// Patch performs an authenticated PATCH request with a JSON body.
func (c *Client) Patch(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.doJSON(ctx, http.MethodPatch, c.baseURL+path, body)
}

// Delete performs an authenticated DELETE request.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, http.MethodDelete, c.baseURL+path, nil)
}

func (c *Client) doJSON(ctx context.Context, method, rawURL string, body any) (*http.Response, error) {
	var buf strings.Builder
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("encoding request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, strings.NewReader(buf.String()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	return resp, nil
}

func (c *Client) do(ctx context.Context, method, rawURL string, body *strings.Reader) (*http.Response, error) {
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, rawURL, body)
	} else {
		req, err = http.NewRequestWithContext(ctx, method, rawURL, http.NoBody)
	}
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	return resp, nil
}

// DecodeError extracts an APIError from a non-2xx response body.
// It supports both { "error": { "code", "message" } } and RFC 9457 { "type", "detail" } shapes.
func DecodeError(resp *http.Response) error {
	if resp == nil {
		return &APIError{Code: "unknown", Message: "missing HTTP response"}
	}
	statusCode, retryAfter := responseErrorMetadata(resp)
	if resp.Body == nil {
		return &APIError{
			Code:       "unknown",
			Message:    fmt.Sprintf("HTTP %d", statusCode),
			StatusCode: statusCode,
			RetryAfter: retryAfter,
		}
	}

	var envelope struct {
		Error  *APIError `json:"error"`
		Type   string    `json:"type"`
		Detail string    `json:"detail"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return &APIError{
			Code:       "unknown",
			Message:    fmt.Sprintf("HTTP %d", statusCode),
			StatusCode: statusCode,
			RetryAfter: retryAfter,
		}
	}
	if envelope.Error != nil {
		envelope.Error.StatusCode = statusCode
		envelope.Error.RetryAfter = retryAfter
		return envelope.Error
	}
	if envelope.Detail != "" {
		code := envelope.Type
		if code == "" {
			code = "problem"
		}
		return &APIError{
			Code:       code,
			Message:    envelope.Detail,
			StatusCode: statusCode,
			RetryAfter: retryAfter,
		}
	}
	return &APIError{
		Code:       "unknown",
		Message:    fmt.Sprintf("HTTP %d", statusCode),
		StatusCode: statusCode,
		RetryAfter: retryAfter,
	}
}

func responseErrorMetadata(resp *http.Response) (statusCode, retryAfter int) {
	statusCode = resp.StatusCode
	retryAfter, err := strconv.Atoi(strings.TrimSpace(resp.Header.Get("Retry-After")))
	if err != nil || retryAfter < 0 {
		retryAfter = 0
	}
	return statusCode, retryAfter
}
