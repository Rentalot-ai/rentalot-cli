package rentalotcli

import "os"

const defaultBaseURL = "https://rentalot.ai"

// Config holds the API connection settings.
type Config struct {
	APIKey  string
	BaseURL string
}

// ConfigFromEnv loads Config from environment variables.
// RENTALOT_API_KEY is required; RENTALOT_BASE_URL defaults to https://rentalot.ai.
func ConfigFromEnv() Config {
	baseURL := os.Getenv("RENTALOT_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return Config{
		APIKey:  os.Getenv("RENTALOT_API_KEY"),
		BaseURL: baseURL,
	}
}
