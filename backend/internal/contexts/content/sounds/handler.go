package sounds

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// Handler serves the ambient sound catalog and audio file bytes.
type Handler struct {
	dir     string
	catalog []Sound
}

// NewHandler builds a handler rooted at dir.
func NewHandler(dir string, catalog []Sound) *Handler {
	return &Handler{dir: dir, catalog: catalog}
}

// ListCatalog returns the sound library as JSON.
func (h *Handler) ListCatalog(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string][]Sound{"sounds": h.catalog})
}

// GetFile serves a sound file.
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	file := chi.URLParam(r, "*")
	path, ok := h.resolve(file)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "invalid sound file")
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, path)
}

// validateFile reports whether file is a safe, single-segment sound file name.
// It rejects empty names and anything that could escape the sound root.
func validateFile(file string) bool {
	if file == "" || file == "." || strings.Contains(file, "..") || strings.ContainsAny(file, `/\`) {
		return false
	}
	return true
}

// resolve validates file and returns the absolute path under the sound root.
func (h *Handler) resolve(file string) (string, bool) {
	if !validateFile(file) {
		return "", false
	}
	path := filepath.Join(h.dir, file)
	rel, err := filepath.Rel(h.dir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return path, true
}
