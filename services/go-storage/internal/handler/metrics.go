package handler

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// Metrics holds process-lifetime atomic counters exposed at GET /metrics.
// All writes use atomic operations so there is no lock contention on hot paths.
type Metrics struct {
	UploadsTotal     atomic.Int64 // single-file uploads attempted
	UploadsFailed    atomic.Int64 // single-file uploads that returned an error
	BytesWritten     atomic.Int64 // bytes committed to final storage (uploads + assemblies)
	SessionsCreated  atomic.Int64 // chunked upload sessions initiated
	SessionsComplete atomic.Int64 // chunked upload sessions assembled successfully
	SessionsAborted  atomic.Int64 // chunked upload sessions explicitly aborted
	DedupHits        atomic.Int64 // CAS hits: file already existed — zero disk write
	DedupMisses      atomic.Int64 // CAS misses: new blob written to content store
}

// metricsHandler returns the http.HandlerFunc that serialises the current counter
// snapshot as a flat JSON object. activeFunc is called at render time to include
// the real-time active-upload count from the limiter without a circular dependency.
func (m *Metrics) metricsHandler(activeFunc func() int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{ //nolint:errcheck
			"uploads_total":     m.UploadsTotal.Load(),
			"uploads_failed":    m.UploadsFailed.Load(),
			"bytes_written":     m.BytesWritten.Load(),
			"sessions_created":  m.SessionsCreated.Load(),
			"sessions_complete": m.SessionsComplete.Load(),
			"sessions_aborted":  m.SessionsAborted.Load(),
			"dedup_hits":        m.DedupHits.Load(),
			"dedup_misses":      m.DedupMisses.Load(),
			"active_uploads":    int64(activeFunc()),
		})
	}
}
