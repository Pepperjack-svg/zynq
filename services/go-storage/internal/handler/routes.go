package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/zynqcloud/go-storage/internal/config"
	"github.com/zynqcloud/go-storage/internal/middleware"
	"github.com/zynqcloud/go-storage/internal/store"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	cfg    *config.Config
	store  store.Backend
	logger *slog.Logger
}

// New registers all routes and returns the root http.Handler.
// Uses Go 1.22 method+path pattern syntax — no external router needed.
func New(cfg *config.Config, backend store.Backend, logger *slog.Logger) http.Handler {
	h := &Handler{cfg: cfg, store: backend, logger: logger}
	auth := middleware.ServiceToken(cfg.ServiceToken)

	mux := http.NewServeMux()

	// ── Single-file streaming upload ─────────────────────────────────────────
	// POST /v1/files
	//   Headers: X-Owner-ID, X-File-ID
	//   Body:    raw (already-encrypted) bytes
	mux.Handle("POST /v1/files", auth(http.HandlerFunc(h.Upload)))

	// ── Streaming download / delete ──────────────────────────────────────────
	mux.Handle("GET /v1/files/{owner}/{fileId}", auth(http.HandlerFunc(h.Download)))
	mux.Handle("DELETE /v1/files/{owner}/{fileId}", auth(http.HandlerFunc(h.Delete)))

	// ── Resumable / chunked upload ───────────────────────────────────────────
	// POST /v1/uploads              → initiate session
	// PUT  /v1/uploads/{id}/parts/{n} → upload part n
	// POST /v1/uploads/{id}/complete  → assemble + finalise
	// DELETE /v1/uploads/{id}         → abort
	mux.Handle("POST /v1/uploads", auth(http.HandlerFunc(h.InitUpload)))
	mux.Handle("PUT /v1/uploads/{sessionId}/parts/{partNum}", auth(http.HandlerFunc(h.UploadPart)))
	mux.Handle("POST /v1/uploads/{sessionId}/complete", auth(http.HandlerFunc(h.CompleteUpload)))
	mux.Handle("DELETE /v1/uploads/{sessionId}", auth(http.HandlerFunc(h.AbortUpload)))

	// ── Health ────────────────────────────────────────────────────────────────
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
