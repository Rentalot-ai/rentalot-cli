package main

import (
	"testing"
)

func TestToQueryParams(t *testing.T) {
	cases := map[string]struct {
		flags    listFlags
		wantKeys map[string]string
		wantLen  int
	}{
		"all zero values": {
			flags:   listFlags{},
			wantLen: 0,
		},
		"limit only": {
			flags:    listFlags{limit: 10},
			wantKeys: map[string]string{"limit": "10"},
			wantLen:  1,
		},
		"page only": {
			flags:    listFlags{page: 2},
			wantKeys: map[string]string{"page": "2"},
			wantLen:  1,
		},
		"filter key=value": {
			flags:    listFlags{filter: "status=active"},
			wantKeys: map[string]string{"status": "active"},
			wantLen:  1,
		},
		"filter without equals": {
			flags:    listFlags{filter: "search-term"},
			wantKeys: map[string]string{"filter": "search-term"},
			wantLen:  1,
		},
		"filter with multiple equals": {
			flags:    listFlags{filter: "key=val=ue"},
			wantKeys: map[string]string{"key": "val=ue"},
			wantLen:  1,
		},
		"all flags set": {
			flags:    listFlags{limit: 25, page: 3, filter: "type=tenant"},
			wantKeys: map[string]string{"limit": "25", "page": "3", "type": "tenant"},
			wantLen:  3,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := tc.flags.toQueryParams()
			if len(got) != tc.wantLen {
				t.Errorf("len(params) = %d, want %d", len(got), tc.wantLen)
			}
			for k, want := range tc.wantKeys {
				if got[k] != want {
					t.Errorf("params[%q] = %q, want %q", k, got[k], want)
				}
			}
		})
	}
}
