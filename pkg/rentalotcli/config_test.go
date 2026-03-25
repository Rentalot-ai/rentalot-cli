package rentalotcli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Missing(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/rentalot/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "" || cfg.BaseURL != "" {
		t.Errorf("expected empty config, got %+v", cfg)
	}
}

func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("api_key: ra_test\nbase_url: http://localhost:3000\n"), 0600); err != nil {
		t.Fatalf("writing config: %v", err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "ra_test" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "ra_test")
	}
	if cfg.BaseURL != "http://localhost:3000" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:3000")
	}
}

func TestSaveConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	original := &Config{APIKey: "ra_abc123", BaseURL: "http://localhost:3000"}
	if err := SaveConfig(original, path); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.APIKey != original.APIKey {
		t.Errorf("APIKey mismatch: got %q, want %q", loaded.APIKey, original.APIKey)
	}
	if loaded.BaseURL != original.BaseURL {
		t.Errorf("BaseURL mismatch: got %q, want %q", loaded.BaseURL, original.BaseURL)
	}
}

func TestSaveConfig_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.yaml")
	if err := SaveConfig(&Config{APIKey: "ra_x"}, path); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestEffective_EnvOverridesFile(t *testing.T) {
	t.Setenv("RENTALOT_API_KEY", "ra_env")
	t.Setenv("RENTALOT_BASE_URL", "http://env:9000")

	cfg := Config{APIKey: "ra_file", BaseURL: "http://file:3000"}
	eff := cfg.Effective()

	if eff.APIKey != "ra_env" {
		t.Errorf("APIKey = %q, want env value", eff.APIKey)
	}
	if eff.BaseURL != "http://env:9000" {
		t.Errorf("BaseURL = %q, want env value", eff.BaseURL)
	}
}

func TestEffective_DefaultBaseURL(t *testing.T) {
	t.Setenv("RENTALOT_API_KEY", "")
	t.Setenv("RENTALOT_BASE_URL", "")

	eff := Config{}.Effective()
	if eff.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", eff.BaseURL, defaultBaseURL)
	}
}

func TestEffective_FileBaseURLPreserved(t *testing.T) {
	t.Setenv("RENTALOT_BASE_URL", "")

	cfg := Config{BaseURL: "http://localhost:3000"}
	eff := cfg.Effective()
	if eff.BaseURL != "http://localhost:3000" {
		t.Errorf("BaseURL = %q, want file value", eff.BaseURL)
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("RENTALOT_API_KEY", "ra_env_only")
	t.Setenv("RENTALOT_BASE_URL", "")

	cfg := ConfigFromEnv()
	if cfg.APIKey != "ra_env_only" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "ra_env_only")
	}
	if cfg.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL = %q, want default", cfg.BaseURL)
	}
}

func TestConfigPath(t *testing.T) {
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
	if filepath.Base(path) != "config.yaml" {
		t.Errorf("expected config.yaml, got %q", filepath.Base(path))
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
