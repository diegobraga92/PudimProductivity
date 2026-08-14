package media

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// maxUploadBytes caps uploads at 10 MB — matches the web client's limit and
// the S3 bucket lifecycle backstop.
const maxUploadBytes = 10 << 20

// MediaHandler serves the direct PUT upload and GET read endpoints for the
// local filesystem media backend.
type MediaHandler struct {
	dir string
}

// NewMediaHandler builds a handler rooted at dir.
func NewMediaHandler(dir string) *MediaHandler {
	return &MediaHandler{dir: dir}
}

// Put writes the request body to <dir>/<key>, capping the size and validating
// the key so the resolved path stays inside the media root.
func (h *MediaHandler) Put(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	path, ok := h.resolve(key)
	if !ok {
		shared.WriteError(w, http.StatusBadRequest, "invalid media key")
		return
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Error().Err(err).Str("key", key).Msg("media: create key directory failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to store media")
		return
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		log.Error().Err(err).Str("key", key).Msg("media: open key for write failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to store media")
		return
	}
	defer f.Close()

	body := http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if _, err := io.Copy(f, body); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			shared.WriteError(w, http.StatusRequestEntityTooLarge, "upload too large (max 10 MB)")
			return
		}
		log.Warn().Err(err).Str("key", key).Msg("media: upload body read failed")
		shared.WriteError(w, http.StatusInternalServerError, "failed to store media")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Get serves the stored object (content-type inferred from the extension).
func (h *MediaHandler) Get(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	path, ok := h.resolve(key)
	if !ok {
		shared.WriteError(w, http.StatusBadRequest, "invalid media key")
		return
	}
	http.ServeFile(w, r, path)
}

// resolve validates key and returns the absolute path under the media root.
func (h *MediaHandler) resolve(key string) (string, bool) {
	if err := ValidateKey(key); err != nil {
		return "", false
	}
	path := filepath.Join(h.dir, filepath.FromSlash(key))
	rel, err := filepath.Rel(h.dir, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return path, true
}
