package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("REDIS_URL")
	os.Unsetenv("HTTP_PORT")

	cfg := Load()

	if cfg.HTTPPort != "8080" {
		t.Errorf("expected default HTTPPort 8080, got %s", cfg.HTTPPort)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("expected default RedisURL, got %s", cfg.RedisURL)
	}
	if cfg.MQTTBrokerURL != "tcp://localhost:1883" {
		t.Errorf("expected default MQTTBrokerURL, got %s", cfg.MQTTBrokerURL)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	os.Setenv("HTTP_PORT", "9090")
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")
	defer func() {
		os.Unsetenv("HTTP_PORT")
		os.Unsetenv("DATABASE_URL")
	}()

	cfg := Load()

	if cfg.HTTPPort != "9090" {
		t.Errorf("expected HTTPPort 9090, got %s", cfg.HTTPPort)
	}
	if cfg.DatabaseURL != "postgres://test:test@localhost/test" {
		t.Errorf("expected DatabaseURL override, got %s", cfg.DatabaseURL)
	}
}

func TestAddress(t *testing.T) {
	cfg := &Config{HTTPPort: "3000"}
	if cfg.Address() != ":3000" {
		t.Errorf("expected :3000, got %s", cfg.Address())
	}
}
