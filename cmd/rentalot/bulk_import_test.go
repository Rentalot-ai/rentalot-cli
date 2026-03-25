package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadImportFile_CSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	content := "name,email,phone\nAlice,alice@example.com,555-0001\nBob,bob@example.com,555-0002\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := loadImportFile(path)
	if err != nil {
		t.Fatalf("loadImportFile() error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("records[0][name] = %v, want Alice", records[0]["name"])
	}
	if records[1]["email"] != "bob@example.com" {
		t.Errorf("records[1][email] = %v, want bob@example.com", records[1]["email"])
	}
}

func TestLoadImportFile_JSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	content := `[{"name":"Alice","email":"alice@example.com"},{"name":"Bob","email":"bob@example.com"}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := loadImportFile(path)
	if err != nil {
		t.Fatalf("loadImportFile() error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("records[0][name] = %v, want Alice", records[0]["name"])
	}
}

func TestLoadImportFile_JSONUpperCase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.JSON")
	content := `[{"id":"1"}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := loadImportFile(path)
	if err != nil {
		t.Fatalf("loadImportFile() error: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("got %d records, want 1", len(records))
	}
}

func TestLoadImportFile_MissingFile(t *testing.T) {
	_, err := loadImportFile("/nonexistent/file.csv")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadImportFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{invalid`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadImportFile(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadImportFile_EmptyCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")
	content := "name,email\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := loadImportFile(path)
	if err != nil {
		t.Fatalf("loadImportFile() error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestLoadImportFile_CSVTrimHeaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spaces.csv")
	content := " name , email \nAlice,alice@test.com\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := loadImportFile(path)
	if err != nil {
		t.Fatalf("loadImportFile() error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("header trimming: records[0][name] = %v, want Alice", records[0]["name"])
	}
}
