package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ── Request / response types ──────────────────────────────────────────────────

type InitUploadRequest struct {
	OwnerID string `json:"owner_id"`
	FileID  string `json:"file_id"`
}

type InitUploadResponse struct {
	SessionID string `json:"session_id"`
}

type PartUploadResponse struct {
	PartNum int    `json:"part_num"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}

type CompleteUploadRequest struct {
	// Optional. When provided the assembled file's SHA-256 is compared and the
	// upload is rejected on mismatch, preventing silent corruption.
	ExpectedSHA256 string `json:"expected_sha256"`
}

// ── Session helpers ───────────────────────────────────────────────────────────

// sessionDir returns the temporary directory used to stage parts for sessionID.
func (h *Handler) sessionDir(sessionID string) string {
	return filepath.Join(h.cfg.StoragePath, ".uploads", sessionID)
}

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// InitUpload creates a resumable upload session and returns its ID.
//
// POST /v1/uploads
// Body: {"owner_id":"…","file_id":"…"}
func (h *Handler) InitUpload(w http.ResponseWriter, r *http.Request) {
	var req InitUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !isValidID(req.OwnerID) || !isValidID(req.FileID) {
		writeError(w, http.StatusBadRequest, "invalid owner_id or file_id")
		return
	}

	sessionID := newSessionID()
	dir := h.sessionDir(sessionID)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		h.logger.Error("init upload: mkdir failed", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}

	// Persist owner/file mapping so CompleteUpload can find the final destination.
	meta := fmt.Sprintf("%s\n%s\n", req.OwnerID, req.FileID)
	if err := os.WriteFile(filepath.Join(dir, "meta"), []byte(meta), 0o640); err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusInternalServerError, "failed to write session metadata")
		return
	}

	h.logger.Info("upload session created", "session", sessionID,
		"owner", req.OwnerID, "file", req.FileID)
	writeJSON(w, http.StatusCreated, InitUploadResponse{SessionID: sessionID})
}

// UploadPart streams a single chunk to disk.
// Parts are numbered from 1; up to 10 000 parts are supported (≈50 TB at 5 GB/part).
//
// PUT /v1/uploads/{sessionId}/parts/{partNum}
// Body: raw bytes for this part
func (h *Handler) UploadPart(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	partNumStr := r.PathValue("partNum")

	if !isValidID(sessionID) {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}
	partNum, err := strconv.Atoi(partNumStr)
	if err != nil || partNum < 1 || partNum > 10_000 {
		writeError(w, http.StatusBadRequest, "partNum must be an integer 1–10000")
		return
	}

	dir := h.sessionDir(sessionID)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	partPath := filepath.Join(dir, fmt.Sprintf("part_%05d", partNum))
	hasher := sha256.New()

	f, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to open part file")
		return
	}

	n, werr := io.Copy(f, io.TeeReader(r.Body, hasher))
	cerr := f.Close()

	if werr != nil || cerr != nil {
		os.Remove(partPath)
		writeError(w, http.StatusInternalServerError, "part write failed")
		return
	}

	writeJSON(w, http.StatusOK, PartUploadResponse{
		PartNum: partNum,
		Size:    n,
		SHA256:  hex.EncodeToString(hasher.Sum(nil)),
	})
}

// CompleteUpload assembles all uploaded parts in order, hashes the result,
// writes the assembled file to its final storage path, and cleans up the session.
//
// POST /v1/uploads/{sessionId}/complete
// Body (optional): {"expected_sha256":"…"}
func (h *Handler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	if !isValidID(sessionID) {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}

	var req CompleteUploadRequest
	json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck

	dir := h.sessionDir(sessionID)
	metaBytes, err := os.ReadFile(filepath.Join(dir, "meta"))
	if err != nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	lines := strings.SplitN(strings.TrimSpace(string(metaBytes)), "\n", 2)
	if len(lines) != 2 {
		writeError(w, http.StatusInternalServerError, "corrupt session metadata")
		return
	}
	ownerID, fileID := lines[0], lines[1]

	// Collect and sort part paths lexicographically (part_00001, part_00002, …).
	entries, err := os.ReadDir(dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read session dir")
		return
	}
	var parts []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "part_") {
			parts = append(parts, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(parts)

	if len(parts) == 0 {
		writeError(w, http.StatusBadRequest, "no parts uploaded")
		return
	}

	// Stream all parts in sequence through a hasher into the storage backend.
	// A pipe connects the goroutine that reads part files to the store.Write call —
	// no intermediate buffer accumulates the full file.
	hasher := sha256.New()
	pr, pw := io.Pipe()

	go func() {
		for _, p := range parts {
			f, err := os.Open(p)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("open part %s: %w", p, err))
				return
			}
			if _, err := io.Copy(pw, f); err != nil {
				f.Close()
				pw.CloseWithError(fmt.Errorf("copy part %s: %w", p, err))
				return
			}
			f.Close()
		}
		pw.Close()
	}()

	finalPath := filepath.Join(ownerID, fileID+".enc")
	n, err := h.store.Write(finalPath, io.TeeReader(pr, hasher))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "assemble failed")
		return
	}

	hash := hex.EncodeToString(hasher.Sum(nil))

	if req.ExpectedSHA256 != "" && req.ExpectedSHA256 != hash {
		h.store.Delete(finalPath) //nolint:errcheck
		writeError(w, http.StatusBadRequest, "sha256 mismatch: upload rejected")
		return
	}

	os.RemoveAll(dir) // best-effort cleanup; failures are non-fatal

	h.logger.Info("chunked upload complete",
		"path", finalPath, "parts", len(parts), "bytes", n, "sha256", hash)

	writeJSON(w, http.StatusCreated, UploadResponse{
		StoragePath: finalPath,
		Size:        n,
		SHA256:      hash,
	})
}

// AbortUpload removes an in-progress upload session and all its staged parts.
//
// DELETE /v1/uploads/{sessionId}
func (h *Handler) AbortUpload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")
	if !isValidID(sessionID) {
		writeError(w, http.StatusBadRequest, "invalid session id")
		return
	}
	os.RemoveAll(h.sessionDir(sessionID)) //nolint:errcheck
	w.WriteHeader(http.StatusNoContent)
}
