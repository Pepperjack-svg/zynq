package config

import (
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

func Load() *Config {
	return &Config{
		Port:                 getEnv("STORAGE_PORT", "5000"),
		StoragePath:          getEnv("STORAGE_PATH", "/data/files"),
		ServiceToken:         getEnv("SERVICE_TOKEN", ""),
		MaxConcurrentUploads: getEnvInt("MAX_CONCURRENT_UPLOADS", 256),
		SessionTTLHours:      getEnvInt("SESSION_TTL_HOURS", 24),
		MinFreeBytes:         getEnvInt64("MIN_FREE_BYTES", 512*1024*1024),
		MaxAssemblyWorkers:   getEnvInt("MAX_ASSEMBLY_WORKERS", 32),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 0 {
			return n
		}
	}
	return fallback
}
