// Package config loads runtime configuration from environment variables.
//
// In Story 1.1, only the bare-minimum fields are populated. Stories that add
// new dependencies (Story 1.5 auth → JwtSecret; Story 6.x payOS → PayosClientID, etc.)
// extend this struct and `Load()` accordingly.
package config

import "os"

// Config holds typed runtime config.
type Config struct {
	Port        string
	AppBaseURL  string
	DatabaseURL string
}

// Load reads env vars and returns a populated Config. Defaults are dev-friendly.
// Stories that introduce required vars MUST add fail-fast validation here.
func Load() *Config {
	return &Config{
		Port:        getenv("PORT", "8080"),
		AppBaseURL:  getenv("APP_BASE_URL", "http://localhost:5173"),
		DatabaseURL: os.Getenv("DATABASE_URL"), // unused until Story 1.2 migrations land
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
