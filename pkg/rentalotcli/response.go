package rentalotcli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DataResponse is the v1 API's data envelope. T can be a resource or a slice.
type DataResponse[T any] struct {
	Data T `json:"data"`
}

// DecodeResponse decodes one JSON value into T and closes the response body.
// Use DataResponse[T] to retain the API's data envelope, or a custom type for
// other response shapes. Unknown fields are allowed for forward compatibility.
// Non-2xx responses return DecodeError's *APIError. A 204 or 205 returns T's
// zero value; an empty body on any other successful status is a decoding error.
func DecodeResponse[T any](resp *http.Response) (T, error) {
	var zero T
	if resp == nil {
		return zero, DecodeError(nil)
	}
	if resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return zero, DecodeError(resp)
	}
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusResetContent {
		return zero, nil
	}
	if resp.Body == nil {
		return zero, fmt.Errorf("decoding response: %w", io.EOF)
	}
	var value T
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&value); err != nil {
		return zero, fmt.Errorf("decoding response: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return zero, fmt.Errorf("decoding response: multiple JSON values")
		}
		return zero, fmt.Errorf("decoding response: %w", err)
	}
	return value, nil
}
