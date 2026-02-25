package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// UploadResponse is returned after a successful single-file or chunked upload.
type UploadResponse struct {
	StoragePath string `json:"storage_path"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
}

// Upload handles a streaming single-file write.
//
// The request body is piped directly through a SHA-256 hasher into the storage
// backend — the full file is never held in memory.
//
// Required headers:
//
//	X-Owner-ID  owner UUID
//	X-File-ID   file UUID
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	ownerID := strings.TrimSpace(r.Header.Get("X-Owner-ID"))
	fileID := strings.TrimSpace(r.Header.Get("X-File-ID"))

	if ownerID == "" || fileID == "" {
		writeError(w, http.StatusBadRequest, "X-Owner-ID and X-File-ID headers are required")
		return
	}
	if !isValidID(ownerID) || !isValidID(fileID) {
		writeError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	storagePath := filepath.Join(ownerID, fileID+".enc")
	hasher := sha256.New()

	// TeeReader: every byte read from r.Body is also written into hasher.
	// The backend streams from tee directly to disk — zero full-file buffering.
	n, err := h.store.Write(storagePath, io.TeeReader(r.Body, hasher))
	if err != nil {
		h.logger.Error("upload: write failed", "path", storagePath, "err", err)
		writeError(w, http.StatusInternalServerError, "storage write failed")
		return
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	h.logger.Info("upload complete", "path", storagePath, "bytes", n, "sha256", hash)

	writeJSON(w, http.StatusCreated, UploadResponse{
		StoragePath: storagePath,
		Size:        n,
		SHA256:      hash,
	})
}

// Download streams a stored file back to the caller without loading it into memory.
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	ownerID := r.PathValue("owner")
	fileID := r.PathValue("fileId")

	if !isValidID(ownerID) || !isValidID(fileID) {
		writeError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	rc, size, err := h.store.Read(filepath.Join(ownerID, fileID+".enc"))
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	io.Copy(w, rc) //nolint:errcheck
}

// Delete permanently removes a file from storage.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ownerID := r.PathValue("owner")
	fileID := r.PathValue("fileId")

	if !isValidID(ownerID) || !isValidID(fileID) {
		writeError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := h.store.Delete(filepath.Join(ownerID, fileID+".enc")); err != nil {
		h.logger.Error("delete failed", "owner", ownerID, "file", fileID, "err", err)
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// isValidID rejects empty values and obvious path-traversal attempts.
// The store.Local backend applies a second, filesystem-level defence via filepath.Clean.
func isValidID(id string) bool {
	return id != "" &&
		!strings.Contains(id, "/") &&
		!strings.Contains(id, "\\") &&
		!strings.Contains(id, "..")
}
