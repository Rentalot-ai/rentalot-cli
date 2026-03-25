package main

import (
	"testing"
)

func TestSourceLabel(t *testing.T) {
	cases := map[string]struct {
		isCustom bool
		want     string
	}{
		"custom config": {
			isCustom: true,
			want:     "file",
		},
		"default config": {
			isCustom: false,
			want:     "default",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := sourceLabel(tc.isCustom)
			if got != tc.want {
				t.Errorf("sourceLabel(%v) = %q, want %q", tc.isCustom, got, tc.want)
			}
		})
	}
}
