package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/diegobraga92/pudimproductivity/backend/internal/eventbus"
	"github.com/diegobraga92/pudimproductivity/backend/internal/featureflag"
	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// Feature flag gating the optional LLM summary (off by default).
const llmFeatureFlag = "insights.llm_enabled"

// LLMSummarizer produces a short natural-language paragraph from a rendered
// report. NoopSummarizer is the default (flag off or no API key).
type LLMSummarizer interface {
	Summarize(ctx context.Context, text string) (string, error)
}

// NoopSummarizer implements LLMSummarizer without calling any external service.
type NoopSummarizer struct{}

func (NoopSummarizer) Summarize(_ context.Context, _ string) (string, error) {
	return "", nil
}

// InsightService aggregates domain events into weekly reports.
//
// It consumes pomodoro.session.completed events to persist focus history (the
// pomodoro module itself is in-memory) and generates template-rendered weekly
// reports on demand. An optional LLM summary is gated behind the
// insights.llm_enabled feature flag.
type InsightService struct {
	repo  InsightRepository
	bus   eventbus.Bus
	flags *featureflag.Service
	llm   LLMSummarizer
	tmpl  *template.Template
}

const reportTemplate = `Week of {{.WeekStart}} summary

- {{.TotalCompletions}} tasks completed ({{printf "%.1f" .CompletionsPerDay}} per day)
{{- if .TopHabits}}
- Top habits: {{range .TopHabits}}{{.Title}} ({{.Count}}x) {{end}}
{{- else}}
- No habit completions this week.
{{- end}}
- {{.FocusMinutes}} focus minutes across {{.FocusSessions}} pomodoro session(s)
- {{.RecipesCreated}} new recipe(s) added`

// NewInsightService builds the insights service.
func NewInsightService(repo InsightRepository, bus eventbus.Bus, flags *featureflag.Service, llm LLMSummarizer) *InsightService {
	if llm == nil {
		llm = NoopSummarizer{}
	}
	tmpl, err := template.New("weekly").Parse(reportTemplate)
	if err != nil {
		// Static template — cannot fail at runtime.
		panic(fmt.Sprintf("parse insights template: %v", err))
	}
	return &InsightService{repo: repo, bus: bus, flags: flags, llm: llm, tmpl: tmpl}
}

// Start subscribes to the events the insights module consumes. The handler
// runs on its own goroutine (with a detached context) so a slow DB write never
// blocks the bus and outlives the publisher's request context.
func (s *InsightService) Start(ctx context.Context) error {
	if s.bus == nil {
		return nil
	}
	_, err := s.bus.Subscribe(ctx, func(_ context.Context, event eventbus.Event) error {
		if event.Type != eventbus.EventPomodoroSessionCompleted {
			return nil
		}
		go func() {
			record := sessionRecordFromEvent(event)
			if record == nil {
				return
			}
			if err := s.repo.UpsertSession(context.Background(), record); err != nil {
				log.Warn().Err(err).Str("session_id", record.ID).Msg("failed to persist pomodoro session")
			}
		}()
		return nil
	})
	return err
}

// toInt coerces a JSON-decoded number (int, float64, json.Number) into an int.
func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
		if f, err := n.Float64(); err == nil {
			return int(f)
		}
	}
	return 0
}

// sessionRecordFromEvent converts a pomodoro.session.completed payload into a
// persistence record. Returns nil when the payload is malformed.
func sessionRecordFromEvent(event eventbus.Event) *PomodoroSessionRecord {
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		return nil
	}
	id, _ := payload["session_id"].(string)
	if id == "" {
		return nil
	}
	userID, _ := payload["user_id"].(string)
	focusMinutes := toInt(payload["focus_minutes"])
	elapsedS := toInt(payload["elapsed_s"])
	startedAt, _ := payload["started_at"].(string)
	completedAt, _ := payload["completed_at"].(string)

	start, err1 := time.Parse(time.RFC3339, startedAt)
	complete, err2 := time.Parse(time.RFC3339, completedAt)
	if err1 != nil || err2 != nil {
		// Fall back to the event timestamp for robustness.
		if err2 != nil {
			complete = event.Timestamp
		}
		if err1 != nil {
			start = complete.Add(-time.Duration(focusMinutes) * time.Minute)
		}
	}

	return &PomodoroSessionRecord{
		ID:           id,
		UserID:       userID,
		FocusMinutes: focusMinutes,
		ElapsedS:     elapsedS,
		StartedAt:    start,
		CompletedAt:  complete,
	}
}

// GenerateWeeklyReport computes (or returns the cached copy of) the weekly
// report for the Monday-starting week containing weekStart.
func (s *InsightService) GenerateWeeklyReport(ctx context.Context, userID string, weekStart time.Time) (*InsightReport, error) {
	monday := startOfWeek(weekStart)
	nextMonday := monday.AddDate(0, 0, 7)

	if cached, err := s.repo.GetReport(ctx, userID, monday); err != nil {
		return nil, fmt.Errorf("get cached report: %w", err)
	} else if cached != nil {
		return cached, nil
	}

	stats, err := s.computeStats(ctx, userID, monday, nextMonday)
	if err != nil {
		return nil, err
	}

	text, err := s.render(stats)
	if err != nil {
		return nil, err
	}

	report := &InsightReport{
		ID:         shared.NewUUID(),
		UserID:     userID,
		WeekStart:  monday,
		Stats:      stats,
		ReportText: text,
		CreatedAt:  time.Now().UTC(),
	}

	// Optional LLM summary, gated by feature flag.
	if enabled, err := s.llmEnabled(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to check insights.llm_enabled flag")
	} else if enabled {
		if summary, err := s.llm.Summarize(ctx, text); err != nil {
			log.Warn().Err(err).Msg("llm summary failed, falling back to template only")
		} else if summary != "" {
			report.LLMSummary = &summary
		}
	}

	if err := s.repo.SaveReport(ctx, report); err != nil {
		return nil, fmt.Errorf("save report: %w", err)
	}
	return report, nil
}

func (s *InsightService) computeStats(ctx context.Context, userID string, from, to time.Time) (WeeklyStats, error) {
	completions, err := s.repo.ListCompletions(ctx, from, to)
	if err != nil {
		return WeeklyStats{}, fmt.Errorf("list completions: %w", err)
	}
	sessions, err := s.repo.ListSessions(ctx, userID, from, to)
	if err != nil {
		return WeeklyStats{}, fmt.Errorf("list sessions: %w", err)
	}
	recipes, err := s.repo.CountRecipes(ctx, from, to)
	if err != nil {
		return WeeklyStats{}, fmt.Errorf("count recipes: %w", err)
	}

	stats := WeeklyStats{
		WeekStart:        from.Format("2006-01-02"),
		TotalCompletions: len(completions),
		TopHabits:        topHabits(completions),
		FocusSessions:    len(sessions),
		RecipesCreated:   recipes,
	}
	for _, s := range sessions {
		stats.FocusMinutes += s.FocusMinutes
	}
	if len(completions) > 0 {
		stats.CompletionsPerDay = float64(len(completions)) / 7.0
	}
	return stats, nil
}

// topHabits aggregates completions per task and returns the top 3 by count,
// preferring tasks that look like habits (completion-based tracking).
func topHabits(completions []CompletionRecord) []HabitStat {
	counts := make(map[string]*HabitStat)
	var order []string
	for _, c := range completions {
		stat, ok := counts[c.TaskID]
		if !ok {
			stat = &HabitStat{TaskID: c.TaskID, Title: c.Title}
			counts[c.TaskID] = stat
			order = append(order, c.TaskID)
		}
		stat.Count++
	}

	stats := make([]HabitStat, 0, len(counts))
	for _, id := range order {
		stats = append(stats, *counts[id])
	}
	// Stable insertion sort by count desc (n is tiny).
	for i := 1; i < len(stats); i++ {
		for j := i; j > 0 && stats[j].Count > stats[j-1].Count; j-- {
			stats[j], stats[j-1] = stats[j-1], stats[j]
		}
	}
	if len(stats) > 3 {
		stats = stats[:3]
	}
	return stats
}

func (s *InsightService) render(stats WeeklyStats) (string, error) {
	var buf bytes.Buffer
	if err := s.tmpl.Execute(&buf, stats); err != nil {
		return "", fmt.Errorf("render report: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func (s *InsightService) llmEnabled(ctx context.Context) (bool, error) {
	if s.flags == nil {
		return false, nil
	}
	return s.flags.IsEnabled(ctx, llmFeatureFlag)
}

// startOfWeek returns the Monday (UTC) of the week containing t.
func startOfWeek(t time.Time) time.Time {
	utc := t.UTC()
	// Weekday: Sunday=0. Offset to Monday.
	offset := (int(utc.Weekday()) + 6) % 7
	return utc.AddDate(0, 0, -offset).Truncate(24 * time.Hour)
}
