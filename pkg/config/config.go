package config

import (
	"os"
	"time"
)

// Config holds runtime configuration for preview-operator.
type Config struct {
	Port         string
	TargetURL    string
	TargetNS     string
	TargetDeploy string
	IdleTimeout  time.Duration
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// Load loads configuration from environment variables.
func Load() Config {
	return Config{
		Port:         getEnv("PORT", "8080"),
		TargetURL:    getEnv("TARGET_URL", "http://sample-app.preview-sample.svc.cluster.local:8080"),
		TargetNS:     getEnv("TARGET_NS", "preview-sample"),
		TargetDeploy: getEnv("TARGET_DEPLOY", "sample-app"),
		IdleTimeout:  5 * time.Minute,
	}
}
