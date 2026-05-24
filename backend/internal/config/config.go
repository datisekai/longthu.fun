// Package config loads runtime configuration from environment variables.
//
// In Story 1.1 only Port + AppBaseURL were populated. Story 1.5 extends with
// DatabaseURL + JWTSecret (both required for the first authenticated request).
// Stories that introduce new required vars MUST add fail-fast validation here.
package config

import (
	"fmt"
	"os"
)

// Config holds typed runtime config.
type Config struct {
	Port        string
	AppBaseURL  string
	DatabaseURL string
	JWTSecret   string
}

// Load reads env vars and returns a populated Config. Returns an error on
// missing required vars (caller fails fast at boot in main.go).
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getenv("PORT", "8080"),
		AppBaseURL:  getenv("APP_BASE_URL", "http://localhost:5173"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config.Load: DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("config.Load: JWT_SECRET is required (generate via openssl rand -hex 32)")
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
