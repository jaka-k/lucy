// Package config loads runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
)

// Config holds runtime settings for the server.
type Config struct {
	APIKey     string
	Port       string
	MongoURI   string
	MongoDB    string
}

// Load reads configuration from the environment, first loading a .env file in
// the working directory if present. The API key and MongoDB connection are
// required; the app fails fast if either is missing.
func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	key := firstNonEmpty(os.Getenv("GEMINI_API_KEY"), os.Getenv("GOOGLE_API_KEY"))
	if key == "" {
		return Config{}, fmt.Errorf("missing API key: set GEMINI_API_KEY (or GOOGLE_API_KEY)")
	}

	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		return Config{}, fmt.Errorf("missing MONGODB_URI")
	}

	mongoDB := os.Getenv("MONGODB_DB")
	if mongoDB == "" {
		return Config{}, fmt.Errorf("missing MONGODB_DB")
	}

	return Config{
		APIKey:   key,
		Port:     firstNonEmpty(os.Getenv("PORT"), "8080"),
		MongoURI: mongoURI,
		MongoDB:  mongoDB,
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
