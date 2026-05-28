package pomodoro

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

type Handler struct {
	service *PomodoroService
}

func NewHandler(service *PomodoroService) *Handler {
	return &Handler{service: service}
}

// DTOs
type SessionResponse struct {
	ID               string       `json:"id"`
	Status           string       `json:"status"`
	FocusDuration    int          `json:"focus_duration"` // minutes
	BreakDuration    int          `json:"break_duration"` // minutes
	CurrentCycle     int          `json:"current_cycle"`
	ElapsedSeconds   int          `json:"elapsed_seconds"`
	RemainingSeconds int          `json:"remaining_seconds"`
	StartedAt        string       `json:"started_at"`
	PausedAt         *string      `json:"paused_at,omitempty"`
	CompletedAt      *string      `json:"completed_at,omitempty"`
	NoiseConfig      *NoiseConfig `json:"noise_config,omitempty"`
}

type startSessionRequest struct {
	FocusDuration int          `json:"focus_duration"` // minutes
	BreakDuration int          `json:"break_duration"` // minutes
	NoiseConfig   *NoiseConfig `json:"noise_config,omitempty"`
}

func toSessionResponse(s *PomodoroSession) SessionResponse {
	resp := SessionResponse{
		ID:               s.ID,
		Status:           string(s.Status),
		FocusDuration:    int(s.FocusDuration.Minutes()),
		BreakDuration:    int(s.BreakDuration.Minutes()),
		CurrentCycle:     s.CurrentCycle,
		ElapsedSeconds:   int(s.Elapsed().Seconds()),
		RemainingSeconds: int(s.Remaining().Seconds()),
		StartedAt:        s.StartedAt.Format(time.RFC3339),
		NoiseConfig:      s.NoiseConfig,
	}
	if s.PausedAt != nil {
		t := s.PausedAt.Format(time.RFC3339)
		resp.PausedAt = &t
	}
	if s.CompletedAt != nil {
		t := s.CompletedAt.Format(time.RFC3339)
		resp.CompletedAt = &t
	}
	return resp
}

// POST /api/v1/pomodoro/start
func (h *Handler) StartSession(w http.ResponseWriter, r *http.Request) {
	var req startSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.service.StartSession(r.Context(), req.FocusDuration, req.BreakDuration, req.NoiseConfig)
	if err != nil {
		log.Error().Err(err).Msg("failed to start pomodoro session")
		shared.WriteError(w, http.StatusInternalServerError, "failed to start session")
		return
	}

	shared.WriteJSON(w, http.StatusCreated, toSessionResponse(session))
}

// GET /api/v1/pomodoro/current
func (h *Handler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	session := h.service.GetCurrent()
	if session == nil {
		shared.WriteJSON(w, http.StatusOK, map[string]interface{}{"active": false})
		return
	}

	resp := toSessionResponse(session)

	shared.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"active":  true,
		"session": resp,
	})
}

// POST /api/v1/pomodoro/pause
func (h *Handler) Pause(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.Pause(r.Context())
	if err != nil {
		shared.WriteError(w, http.StatusConflict, err.Error())
		return
	}

	shared.WriteJSON(w, http.StatusOK, toSessionResponse(session))
}

// POST /api/v1/pomodoro/resume
func (h *Handler) Resume(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.Resume(r.Context())
	if err != nil {
		shared.WriteError(w, http.StatusConflict, err.Error())
		return
	}

	shared.WriteJSON(w, http.StatusOK, toSessionResponse(session))
}

// POST /api/v1/pomodoro/stop
func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	session, err := h.service.Stop(r.Context())
	if err != nil {
		shared.WriteError(w, http.StatusConflict, err.Error())
		return
	}

	shared.WriteJSON(w, http.StatusOK, toSessionResponse(session))
}
