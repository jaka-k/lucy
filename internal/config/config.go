// Package config loads runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
)

const defaultModel = "gemini-2.5-flash"

// Config holds runtime settings for the server.
type Config struct {
	APIKey string
	Port   string
	Model  string
}

// Load reads configuration from the environment, first loading a .env file in
// the working directory if present. The API key is required and is read from
// GEMINI_API_KEY, falling back to GOOGLE_API_KEY.
func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	key := firstNonEmpty(os.Getenv("GEMINI_API_KEY"), os.Getenv("GOOGLE_API_KEY"))
	if key == "" {
		return Config{}, fmt.Errorf("missing API key: set GEMINI_API_KEY (or GOOGLE_API_KEY)")
	}

	return Config{
		APIKey: key,
		Port:   firstNonEmpty(os.Getenv("PORT"), "8080"),
		Model:  firstNonEmpty(os.Getenv("GEMINI_MODEL"), defaultModel),
	}, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
