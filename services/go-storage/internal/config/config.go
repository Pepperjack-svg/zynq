package config

import "os"

// Config holds all runtime configuration for the storage service.
type Config struct {
	Port         string
	StoragePath  string
	ServiceToken string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("STORAGE_PORT", "5000"),
		StoragePath:  getEnv("STORAGE_PATH", "/data/files"),
		ServiceToken: getEnv("SERVICE_TOKEN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
