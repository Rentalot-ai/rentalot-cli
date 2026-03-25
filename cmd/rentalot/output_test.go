package main

import (
	"strings"
	"testing"
)

func TestStr(t *testing.T) {
	cases := map[string]struct {
		m    map[string]any
		key  string
		want string
	}{
		"existing string": {
			m:    map[string]any{"name": "Alice"},
			key:  "name",
			want: "Alice",
		},
		"existing number": {
			m:    map[string]any{"count": 42},
			key:  "count",
			want: "42",
		},
		"missing key": {
			m:    map[string]any{"name": "Alice"},
			key:  "missing",
			want: "",
		},
		"nil value": {
			m:    map[string]any{"name": nil},
			key:  "name",
			want: "",
		},
		"empty map": {
			m:    map[string]any{},
			key:  "any",
			want: "",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := str(tc.m, tc.key)
			if got != tc.want {
				t.Errorf("str(%v, %q) = %q, want %q", tc.m, tc.key, got, tc.want)
			}
		})
	}
}

func TestExtractList(t *testing.T) {
	cases := map[string]struct {
		input any
		want  int
	}{
		"direct array": {
			input: []any{
				map[string]any{"id": "1"},
				map[string]any{"id": "2"},
			},
			want: 2,
		},
		"data envelope": {
			input: map[string]any{
				"data": []any{
					map[string]any{"id": "1"},
				},
			},
			want: 1,
		},
		"items envelope": {
			input: map[string]any{
				"items": []any{
					map[string]any{"id": "1"},
					map[string]any{"id": "2"},
					map[string]any{"id": "3"},
				},
			},
			want: 3,
		},
		"results envelope": {
			input: map[string]any{
				"results": []any{
					map[string]any{"id": "1"},
				},
			},
			want: 1,
		},
		"fallback array field": {
			input: map[string]any{
				"contacts": []any{
					map[string]any{"id": "1"},
				},
			},
			want: 1,
		},
		"nil input": {
			input: nil,
			want:  0,
		},
		"string input": {
			input: "not a list",
			want:  0,
		},
		"empty array": {
			input: []any{},
			want:  0,
		},
		"map without arrays": {
			input: map[string]any{"key": "value"},
			want:  0,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := extractList(tc.input)
			if len(got) != tc.want {
				t.Errorf("extractList() returned %d items, want %d", len(got), tc.want)
			}
		})
	}
}

func TestToMaps(t *testing.T) {
	cases := map[string]struct {
		input []any
		want  int
	}{
		"all maps": {
			input: []any{
				map[string]any{"a": 1},
				map[string]any{"b": 2},
			},
			want: 2,
		},
		"mixed types skips non-maps": {
			input: []any{
				map[string]any{"a": 1},
				"not a map",
				42,
			},
			want: 1,
		},
		"empty slice": {
			input: []any{},
			want:  0,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := toMaps(tc.input)
			if len(got) != tc.want {
				t.Errorf("toMaps() returned %d items, want %d", len(got), tc.want)
			}
		})
	}
}

func TestDecodeBody(t *testing.T) {
	cases := map[string]struct {
		input   string
		wantErr bool
	}{
		"valid json": {
			input:   `{"key": "value"}`,
			wantErr: false,
		},
		"invalid json": {
			input:   `{invalid`,
			wantErr: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var v any
			err := decodeBody(strings.NewReader(tc.input), &v)
			if (err != nil) != tc.wantErr {
				t.Errorf("decodeBody() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestPrintRawJSON(t *testing.T) {
	cases := map[string]struct {
		input   string
		wantErr bool
	}{
		"valid json": {
			input:   `{"key":"value"}`,
			wantErr: false,
		},
		"invalid json": {
			input:   `{bad`,
			wantErr: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := printRawJSON(strings.NewReader(tc.input))
			if (err != nil) != tc.wantErr {
				t.Errorf("printRawJSON() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestPrintTable(t *testing.T) {
	// Verify printTable doesn't panic with various inputs.
	t.Run("normal", func(t *testing.T) {
		printTable(
			[]string{"ID", "NAME", "EMAIL"},
			[][]string{
				{"1", "Alice", "alice@test.com"},
				{"2", "Bob", "bob@test.com"},
			},
		)
	})
	t.Run("empty rows", func(t *testing.T) {
		printTable([]string{"ID", "NAME"}, [][]string{})
	})
	t.Run("single column", func(t *testing.T) {
		printTable([]string{"ID"}, [][]string{{"1"}, {"2"}})
	})
}
