// Package config provides unified configuration loading for all VoxMesh services.
package config

import (
	"fmt"
	"os"
)

// Config holds common configuration values shared across services.
type Config struct {
	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// MQTT
	MQTTBrokerURL string
	MQTTClientID  string

	// HTTP
	HTTPPort string

	// Service URLs (for inter-service HTTP calls)
	AuthServiceURL    string
	ChannelServiceURL string
	AudioMixerAddr    string // gRPC address for Audio Mixer

	// JWT
	JWTPrivateKeyPath string
	JWTPublicKeyPath  string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379/0"),
		MQTTBrokerURL:     getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTClientID:      getEnv("MQTT_CLIENT_ID", "voxmesh-service"),
		HTTPPort:          getEnv("HTTP_PORT", "8080"),
		AuthServiceURL:    getEnv("AUTH_SERVICE_URL", "http://localhost:8081"),
		ChannelServiceURL: getEnv("CHANNEL_SERVICE_URL", "http://localhost:8082"),
		AudioMixerAddr:    getEnv("AUDIO_MIXER_ADDR", "localhost:9000"),
		JWTPrivateKeyPath: getEnv("JWT_PRIVATE_KEY", "./secrets/jwt_private.pem"),
		JWTPublicKeyPath:  getEnv("JWT_PUBLIC_KEY", "./secrets/jwt_public.pem"),
	}
}

func (c *Config) Address() string {
	return fmt.Sprintf(":%s", c.HTTPPort)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
