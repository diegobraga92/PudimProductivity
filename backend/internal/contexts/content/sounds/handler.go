package sounds

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// Handler serves the ambient sound catalog, the audio file bytes, and the
// endpoints that manage the sounds a user has added.
type Handler struct {
	dir     string
	catalog []Sound
	store   *Store
}

// NewHandler builds a handler rooted at dir. The store backing user-added
// sounds is optional: when dir is not writable, the library still serves the
// built-in catalog and the write endpoints report 503.
func NewHandler(dir string, catalog []Sound) *Handler {
	store, err := NewStore(dir, catalog)
	if err != nil {
		log.Warn().Err(err).Str("dir", dir).Msg("sounds: user-added sounds disabled")
	}
	return &Handler{dir: dir, catalog: catalog, store: store}
}

// ListCatalog returns the sound library as JSON: the built-in sounds followed
// by the sounds the user added.
func (h *Handler) ListCatalog(w http.ResponseWriter, _ *http.Request) {
	sounds := h.catalog
	if h.store != nil {
		sounds = h.store.List()
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string][]Sound{"sounds": sounds})
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

// maxUploadBytes caps the whole multipart request body. It must stay in sync
// with nginx's client_max_body_size and leaves headroom over maxSoundFileBytes
// for the multipart encoding (boundaries and part headers).
const maxUploadBytes = 12 << 20

// maxSoundFileBytes is the user-facing cap on an uploaded audio file.
const maxSoundFileBytes = 10 << 20

// Create stores an uploaded audio file plus its metadata as a new sound. The
// request is multipart/form-data with "label", "icon" (optional) and "file".
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if !h.writable(w) {
		return
	}

	// ParseMultipartForm's argument is the in-memory budget.
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, "sound file is too large (max 10 MB)")
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	defer func() { _ = r.MultipartForm.RemoveAll() }()

	label, icon, err := normalizeMetadata(r.FormValue("label"), r.FormValue("icon"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "an audio file is required")
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > maxSoundFileBytes {
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, "sound file is too large (max 10 MB)")
		return
	}

	mime, ext, ok := detectAudioFormat(file)
	if !ok {
		httpx.WriteError(w, http.StatusUnsupportedMediaType, "unsupported audio format (use MP3, OGG, M4A or WAV)")
		return
	}

	sound, err := h.store.Add(label, icon, file, ext, mime)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, sound)
}

// Update changes the name and icon of a sound the user added.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if !h.writable(w) {
		return
	}
	var req UpdateSoundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	label, icon, err := normalizeMetadata(req.Label, req.Icon)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	sound, err := h.store.Update(chi.URLParam(r, "soundId"), label, icon)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, sound)
}

// Delete removes a sound the user added, along with its audio file.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if !h.writable(w) {
		return
	}
	if err := h.store.Delete(chi.URLParam(r, "soundId")); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writable reports whether the library can be modified, writing 503 when the
// sounds directory was unusable at startup.
func (h *Handler) writable(w http.ResponseWriter) bool {
	if h.store != nil {
		return true
	}
	httpx.WriteError(w, http.StatusServiceUnavailable, "sound library is not writable")
	return false
}

// normalizeMetadata trims and validates the user-supplied name and icon.
func normalizeMetadata(label, icon string) (string, string, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return "", "", errors.New("a name is required")
	}
	if utf8.RuneCountInString(label) > maxLabelLen {
		return "", "", fmt.Errorf("name must be at most %d characters", maxLabelLen)
	}
	icon = strings.TrimSpace(icon)
	if icon == "" {
		icon = defaultSoundIcon
	}
	if utf8.RuneCountInString(icon) > maxIconRunes {
		return "", "", errors.New("icon must be a single emoji")
	}
	return label, icon, nil
}

// writeStoreError maps store errors onto HTTP responses.
func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrDefaultSound):
		httpx.WriteError(w, http.StatusConflict, "built-in sounds cannot be changed")
	case errors.Is(err, ErrUnknownSound):
		httpx.WriteError(w, http.StatusNotFound, "sound not found")
	default:
		log.Error().Err(err).Msg("sounds: store operation failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to update the sound library")
	}
}
