package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the storage service.
// Every field has an environment variable override so the container image
// is identical across environments; only the env vars change.
type Config struct {
	Port         string
	StoragePath  string
	ServiceToken string

	// MaxConcurrentUploads caps goroutines handling active uploads.
	// Each slot costs ~520 KB (512 KB write buffer + goroutine stack + fd).
	// Default 256 → ~133 MB.  Set via MAX_CONCURRENT_UPLOADS.
	MaxConcurrentUploads int

	// SessionTTLHours is how old an incomplete upload session directory must be
	// before the cleanup goroutine removes it. 0 disables cleanup entirely.
	// Set via SESSION_TTL_HOURS.
	SessionTTLHours int

	// MinFreeBytes is the readiness-probe threshold: /healthz/ready returns 503
	// when the storage volume has less than this many bytes available.
	// Default 512 MB.  Set via MIN_FREE_BYTES.
	MinFreeBytes int64

	// MaxAssemblyWorkers caps the number of concurrent CompleteUpload goroutines
	// performing disk I/O (part concatenation + SHA-256 hashing). Assembly is
	// both CPU-bound (hashing) and I/O-bound (reading part files + writing the
	// final blob). Without a cap, many simultaneous completions cause disk
	// thrashing and scheduler starvation.
	// Default 32.  Set via MAX_ASSEMBLY_WORKERS.
	MaxAssemblyWorkers int
}

// Load reads configuration from environment variables and returns an error
// if required values are missing or invalid.
//
// SERVICE_TOKEN is required unless INSECURE_STORAGE=true is explicitly set
// (development/testing only). An empty token allows all requests through
// without authentication, which is unsafe in production.
func Load() (*Config, error) {
	token := os.Getenv("SERVICE_TOKEN")
	if token == "" && os.Getenv("INSECURE_STORAGE") != "true" {
		return nil, fmt.Errorf(
			"SERVICE_TOKEN is not set; set it to a strong random value, " +
				"or set INSECURE_STORAGE=true to disable auth (dev only)",
		)
	}

	return &Config{
		Port:                 getEnv("STORAGE_PORT", "5000"),
		StoragePath:          getEnv("STORAGE_PATH", "/data/files"),
		ServiceToken:         token,
		MaxConcurrentUploads: getEnvInt("MAX_CONCURRENT_UPLOADS", 256),
		SessionTTLHours:      getEnvInt("SESSION_TTL_HOURS", 24),
		MinFreeBytes:         getEnvInt64("MIN_FREE_BYTES", 512*1024*1024),
		MaxAssemblyWorkers:   getEnvInt("MAX_ASSEMBLY_WORKERS", 32),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvInt returns the integer value of key, or fallback if the variable is
// unset or unparseable. A configured value of 0 is accepted (caller may
// interpret it as "disabled").
func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return fallback
}

// getEnvInt64 returns the int64 value of key, or fallback if the variable is
// unset or unparseable. Negative values are rejected (sizes must be ≥ 0).
func getEnvInt64(key string, fallback int64) int64 {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
		return n
	}
	return fallback
}
