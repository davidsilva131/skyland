// Package config carga la configuración de Skyland desde el entorno.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config es la configuración del proceso (API o worker).
type Config struct {
	Environment     string
	DatabaseURL     string
	UpstashRedisURL string
	JWTSecret       string
	CORSOrigins     []string
}

// Load lee las variables SKYLAND_* del entorno y valida las obligatorias.
func Load() (*Config, error) {
	cfg := &Config{
		Environment:     getenv("SKYLAND_ENVIRONMENT", "development"),
		DatabaseURL:     os.Getenv("SKYLAND_DATABASE_URL"),
		UpstashRedisURL: os.Getenv("SKYLAND_UPSTASH_REDIS_URL"),
		JWTSecret:       os.Getenv("SKYLAND_JWT_SECRET"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("SKYLAND_DATABASE_URL es obligatoria")
	}

	if raw := os.Getenv("SKYLAND_CORS_ORIGINS"); raw != "" {
		for _, origin := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(origin); trimmed != "" {
				cfg.CORSOrigins = append(cfg.CORSOrigins, trimmed)
			}
		}
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
