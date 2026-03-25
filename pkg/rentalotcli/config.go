package rentalotcli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultBaseURL = "https://rentalot.ai"

// Config holds the API connection settings.
// It is the on-disk YAML shape and the runtime config.
type Config struct {
	APIKey  string `yaml:"api_key,omitempty"`
	BaseURL string `yaml:"base_url,omitempty"`
}

// ConfigPath returns the default config file path: ~/.config/rentalot/config.yaml.
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting config dir: %w", err)
	}
	return filepath.Join(dir, "rentalot", "config.yaml"), nil
}

// LoadConfig reads a YAML config file from path.
// Returns an empty Config (not an error) if the file doesn't exist.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return &cfg, nil
}

// SaveConfig writes cfg as YAML to path, creating parent directories as needed.
func SaveConfig(cfg *Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}
	return nil
}

// Effective returns the config with environment variable overrides applied.
// RENTALOT_API_KEY and RENTALOT_BASE_URL override file values.
// BaseURL defaults to https://rentalot.ai if unset after env overlay.
func (c Config) Effective() Config {
	if v := os.Getenv("RENTALOT_API_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("RENTALOT_BASE_URL"); v != "" {
		c.BaseURL = v
	}
	if c.BaseURL == "" {
		c.BaseURL = defaultBaseURL
	}
	return c
}

// ConfigFromEnv returns a Config loaded purely from environment variables.
// Equivalent to Config{}.Effective().
func ConfigFromEnv() Config {
	return Config{}.Effective()
}
