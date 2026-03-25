package main

import (
	"testing"

	"github.com/fatih/color"
)

func init() {
	// Disable color output in tests for predictable assertions.
	color.NoColor = true
}

func TestHighlight(t *testing.T) {
	got := highlight("test")
	if got != "test" {
		t.Errorf("highlight(%q) = %q, want %q (with color disabled)", "test", got, "test")
	}
}

func TestFileRef(t *testing.T) {
	got := fileRef("/path/to/file")
	if got != "/path/to/file" {
		t.Errorf("fileRef() = %q, want %q", got, "/path/to/file")
	}
}

func TestVersionRef(t *testing.T) {
	got := versionRef("v1.0.0")
	if got != "v1.0.0" {
		t.Errorf("versionRef() = %q, want %q", got, "v1.0.0")
	}
}

func TestSuccess(t *testing.T) {
	// Verify no panic.
	success("Created %s", "item")
}

func TestWarn(t *testing.T) {
	// Verify no panic.
	warn("Warning: %s", "something")
}
