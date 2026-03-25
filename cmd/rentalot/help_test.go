package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestSplitFlagLine(t *testing.T) {
	cases := map[string]struct {
		input    string
		wantLen  int
		wantFlag string
		wantDesc string
	}{
		"flag with description": {
			input:    "--name string   contact name",
			wantLen:  2,
			wantFlag: "--name string",
			wantDesc: "contact name",
		},
		"flag without description": {
			input:   "--verbose",
			wantLen: 1,
		},
		"short and long flag": {
			input:    "-n, --name string   contact name (required)",
			wantLen:  2,
			wantFlag: "-n, --name string",
			wantDesc: "contact name (required)",
		},
		"empty string": {
			input:   "",
			wantLen: 1,
		},
		"single word": {
			input:   "flag",
			wantLen: 1,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := splitFlagLine(tc.input)
			if len(got) != tc.wantLen {
				t.Errorf("splitFlagLine(%q) returned %d parts, want %d: %v", tc.input, len(got), tc.wantLen, got)
				return
			}
			if tc.wantLen == 2 {
				if got[0] != tc.wantFlag {
					t.Errorf("flag = %q, want %q", got[0], tc.wantFlag)
				}
				if got[1] != tc.wantDesc {
					t.Errorf("desc = %q, want %q", got[1], tc.wantDesc)
				}
			}
		})
	}
}

func TestVisibleSubcommands(t *testing.T) {
	parent := &cobra.Command{Use: "root"}
	visible := &cobra.Command{Use: "list", Short: "List items"}
	hidden := &cobra.Command{Use: "internal", Short: "Internal", Hidden: true}
	parent.AddCommand(visible, hidden)

	got := visibleSubcommands(parent)

	// cobra auto-adds "help", which visibleSubcommands filters out
	for _, cmd := range got {
		if cmd.Name() == "help" {
			t.Error("visibleSubcommands should filter out 'help'")
		}
		if cmd.Hidden {
			t.Error("visibleSubcommands should filter out hidden commands")
		}
	}
	if len(got) != 1 {
		t.Errorf("got %d visible commands, want 1", len(got))
	}
	if got[0].Name() != "list" {
		t.Errorf("got command %q, want %q", got[0].Name(), "list")
	}
}

func TestMaxCommandNameLen(t *testing.T) {
	cases := map[string]struct {
		names []string
		want  int
	}{
		"single": {
			names: []string{"list"},
			want:  4,
		},
		"multiple": {
			names: []string{"list", "create", "delete"},
			want:  6,
		},
		"empty": {
			names: []string{},
			want:  0,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var cmds []*cobra.Command
			for _, n := range tc.names {
				cmds = append(cmds, &cobra.Command{Use: n})
			}
			got := maxCommandNameLen(cmds)
			if got != tc.want {
				t.Errorf("maxCommandNameLen() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestColorizedHelp(t *testing.T) {
	parent := &cobra.Command{
		Use:   "rentalot",
		Short: "CLI tool",
		Long:  "A longer description of the CLI tool",
	}
	sub := &cobra.Command{
		Use:     "contacts",
		Aliases: []string{"c"},
		Short:   "Manage contacts",
		Run:     func(cmd *cobra.Command, args []string) {},
	}
	sub.Flags().String("name", "", "contact name")
	parent.AddCommand(sub)
	parent.PersistentFlags().Bool("json", false, "output as JSON")

	// Test parent help (has subcommands).
	colorizedHelp(parent, nil)

	// Test subcommand help (runnable, has aliases, flags).
	colorizedHelp(sub, nil)
}

func TestPrintColorizedFlags(t *testing.T) {
	flags := "      --name string   contact name\n      --email string   email address\n"
	// Just verify it doesn't panic.
	printColorizedFlags(flags)
}

func TestPrintColorizedFlags_EmptyLine(t *testing.T) {
	flags := "      --name string   contact name\n\n      --email string   email\n"
	printColorizedFlags(flags)
}

func TestPrintColorizedFlags_NoDescription(t *testing.T) {
	flags := "      --verbose\n"
	printColorizedFlags(flags)
}
