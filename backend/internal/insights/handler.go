package insights

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// reportResponse is the API shape for a weekly report.
type reportResponse struct {
	WeekStart   string      `json:"week_start"`
	Stats       WeeklyStats `json:"stats"`
	ReportText  string      `json:"report_text"`
	LLMSummary  *string     `json:"llm_summary,omitempty"`
	GeneratedAt string      `json:"generated_at"`
}

type Handler struct {
	service *InsightService
}

func NewHandler(service *InsightService) *Handler {
	return &Handler{service: service}
}

func toReportResponse(r *InsightReport) reportResponse {
	return reportResponse{
		WeekStart:   r.WeekStart.Format("2006-01-02"),
		Stats:       r.Stats,
		ReportText:  r.ReportText,
		LLMSummary:  r.LLMSummary,
		GeneratedAt: r.CreatedAt.Format(time.RFC3339),
	}
}

// GET /api/v1/insights/weekly?date=YYYY-MM-DD
// date defaults to today; the report covers the Monday-starting week containing
// it. Returns the cached report if one exists.
func (h *Handler) WeeklyReport(w http.ResponseWriter, r *http.Request) {
	weekStart := time.Now().UTC()
	if raw := r.URL.Query().Get("date"); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			shared.WriteError(w, http.StatusBadRequest, "date must be YYYY-MM-DD")
			return
		}
		weekStart = parsed
	}

	userID := shared.GetUserID(r.Context())
	report, err := h.service.GenerateWeeklyReport(r.Context(), userID, weekStart)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("failed to generate weekly report")
		shared.WriteError(w, http.StatusInternalServerError, "failed to generate weekly report")
		return
	}

	shared.WriteJSON(w, http.StatusOK, toReportResponse(report))
}
