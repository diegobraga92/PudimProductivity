package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	httpx "github.com/diegobraga92/pudimproductivity/backend/internal/platform/http"
)

// maxImportBytes caps the size of an upload so a huge/malicious body can't
// exhaust server memory. 50 MiB is far beyond the realistic size of a
// text-only backup for a single-user app.
const maxImportBytes = 50 << 20 // 50 MiB

// Handler is the HTTP transport for the backup module.
type Handler struct {
	service    *Service
	appVersion string
}

func NewHandler(service *Service, appVersion string) *Handler {
	return &Handler{service: service, appVersion: appVersion}
}

// Export handles GET /api/v1/backup/export. It streams a full backup of the
// non-sensitive data as a JSON download.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Export(r.Context(), h.appVersion)
	if err != nil {
		log.Error().Err(err).Msg("backup export failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to export backup")
		return
	}

	filename := fmt.Sprintf("pudim-backup-%s.json", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		log.Error().Err(err).Msg("failed to stream backup export")
	}
}

// Import handles POST /api/v1/backup/import. It validates and restores a
// backup, replacing the current contents of every backed-up table. The restore
// is transactional — a malformed or incompatible backup changes nothing.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "failed to read backup body")
		return
	}
	if len(data) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "backup body is empty")
		return
	}

	// Early syntax check so malformed payloads fail before touching the DB.
	var probe BackupFile
	if err := json.Unmarshal(data, &probe); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid backup: not a valid backup JSON document")
		return
	}

	result, err := h.service.Import(r.Context(), data)
	if err != nil {
		if errors.Is(err, ErrInvalidBackup) || errors.Is(err, ErrUnsupportedVersion) {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Error().Err(err).Msg("backup import failed")
		httpx.WriteError(w, http.StatusInternalServerError, "failed to import backup")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, result)
}
