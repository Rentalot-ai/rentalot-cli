package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/Rentalot-ai/rentalot-cli/pkg/rentalotcli"
)

var jsonOutput bool

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output as JSON")
}

// printRawJSON pretty-prints the JSON from the given reader to stdout.
func printRawJSON(body io.Reader) error {
	var v any
	if err := json.NewDecoder(body).Decode(&v); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// printTable writes a tab-aligned table with headers and rows to stdout.
func printTable(headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, strings.Join(headers, "\t"))
	sep := make([]string, len(headers))
	for i, h := range headers {
		sep[i] = strings.Repeat("-", len(h))
	}
	_, _ = fmt.Fprintln(tw, strings.Join(sep, "\t"))
	for _, row := range rows {
		_, _ = fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	_ = tw.Flush()
}

// str safely extracts a string field from a map.
func str(m map[string]any, key string) string {
	if v, ok := m[key]; ok && v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// extractList tries common response shapes to find the list of items.
// It handles: []any, {"data": [...]}, {"items": [...]}, {"<key>": [...]}.
func extractList(v any) []map[string]any {
	switch t := v.(type) {
	case []any:
		return toMaps(t)
	case map[string]any:
		for _, key := range []string{"data", "items", "results"} {
			if raw, ok := t[key]; ok {
				if arr, ok := raw.([]any); ok {
					return toMaps(arr)
				}
			}
		}
		// Try any array field.
		for _, val := range t {
			if arr, ok := val.([]any); ok {
				return toMaps(arr)
			}
		}
	}
	return nil
}

// decodeAPIError wraps rentalotcli.DecodeError for use in commands.
func decodeAPIError(resp *http.Response) error {
	return rentalotcli.DecodeError(resp)
}

// decodeBody decodes the response body into v.
func decodeBody(body io.Reader, v any) error {
	if err := json.NewDecoder(body).Decode(v); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func toMaps(arr []any) []map[string]any {
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}
