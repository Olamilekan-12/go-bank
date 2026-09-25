package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HTTP_PORT", "")
	os.Unsetenv("HTTP_PORT")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	os.Unsetenv("SHUTDOWN_TIMEOUT")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.HTTPPort != "8080" {
		t.Errorf("HTTPPort = %q want %q", cfg.HTTPPort, "8080")
	}

	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v want %v", cfg.ShutdownTimeout, 30*time.Second)
	}
}

func TestLoadInvalidShutdownTimeout(t *testing.T) {
	t.Setenv("SHUTDOWN_TIMEOUT", "banana")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error for invalid SHUTDOWN_TIMEOUT, got nil")
	}
}
