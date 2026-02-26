package middleware

import (
	"net/http"
	"strconv"
)

const (
	// defaultUploadConcurrency is the fallback slot count when maxConcurrent ≤ 0.
	defaultUploadConcurrency = 256

	// retryAfterSeconds is the value of the Retry-After header sent on 503.
	retryAfterSeconds = "5"

	// capacityErrorPayload is the fixed JSON body returned when the limiter rejects a request.
	capacityErrorPayload = `{"error":"server at capacity — retry in 5s"}`
)

// UploadLimiter caps the number of concurrently active upload goroutines using
// a non-blocking channel semaphore. When the semaphore is full, new requests
// receive HTTP 503 + Retry-After immediately rather than queuing — queuing under
// a 100k-concurrent spike would exhaust RAM before providing any relief.
//
// Sizing guidance (rule of thumb):
//
//	MAX_CONCURRENT_UPLOADS = min(available_RAM_GB * 4, open_file_limit / 4)
//
// Each in-flight upload holds one 512 KB write buffer (local.go) + one goroutine
// (~8 KB stack) + one open fd = ~520 KB per slot. 256 slots ≈ 133 MB.
type UploadLimiter struct {
	sem chan struct{}
}

// NewUploadLimiter creates a limiter allowing at most maxConcurrent simultaneous uploads.
func NewUploadLimiter(maxConcurrent int) *UploadLimiter {
	if maxConcurrent <= 0 {
		maxConcurrent = defaultUploadConcurrency
	}
	return &UploadLimiter{sem: make(chan struct{}, maxConcurrent)}
}

// Limit wraps a handler so that each request must acquire a slot from the
// semaphore before proceeding. Requests that cannot acquire immediately get 503.
func (l *UploadLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case l.sem <- struct{}{}:
			defer func() { <-l.sem }()
			next.ServeHTTP(w, r)
		default:
			// Server at capacity — tell the client to back off.
			w.Header().Set("Retry-After", retryAfterSeconds)
			w.Header().Set("X-Active-Uploads", strconv.Itoa(len(l.sem)))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(capacityErrorPayload)) //nolint:errcheck
		}
	})
}

// Active returns the number of upload slots currently in use.
func (l *UploadLimiter) Active() int { return len(l.sem) }

// Cap returns the maximum number of concurrent upload slots.
func (l *UploadLimiter) Cap() int { return cap(l.sem) }
