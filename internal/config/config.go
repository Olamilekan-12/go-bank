package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Env             string
	HTTPPort        string
	ShutdownTimeout time.Duration
}

func Load() (*Config, error) {
	shutdownTimeout, err := time.ParseDuration(getEnv("SHUTDOWN_TIMEOUT", "30s"))
	if err != nil {
		return nil, fmt.Errorf("invalid SHUTDOWN_TIMEOUT: %w", err)
	}

	cfg := &Config{
		Env:             getEnv("APP_ENV", "development"),
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		ShutdownTimeout: shutdownTimeout,
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
