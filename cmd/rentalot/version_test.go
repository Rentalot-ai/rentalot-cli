package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestTruncateCommit(t *testing.T) {
	cases := map[string]struct {
		input string
		want  string
	}{
		"long commit": {
			input: "abc123def456",
			want:  "abc123de",
		},
		"exactly 8 chars": {
			input: "abc123de",
			want:  "abc123de",
		},
		"short commit": {
			input: "abc",
			want:  "abc",
		},
		"empty": {
			input: "",
			want:  "",
		},
		"dev build": {
			input: "dev",
			want:  "dev",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := truncateCommit(tc.input)
			if got != tc.want {
				t.Errorf("truncateCommit(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestPrintPlainVersion(t *testing.T) {
	// Just verify it doesn't panic.
	printPlainVersion()
}

func TestPrintPrettyVersion(t *testing.T) {
	// Just verify it doesn't panic.
	printPrettyVersion()
}

func TestVersionCmd_Plain(t *testing.T) {
	versionPlain = true
	t.Cleanup(func() { versionPlain = false })
	versionCmd.Run(versionCmd, nil)
}

func TestVersionCmd_Pretty(t *testing.T) {
	versionPlain = false
	versionCmd.Run(versionCmd, nil)
}

func TestVersionCmd_Execute(t *testing.T) {
	cmd := &cobra.Command{Use: "rentalot"}
	v := *versionCmd // copy to avoid mutation
	cmd.AddCommand(&v)
	cmd.SetArgs([]string{"version", "--plain"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
