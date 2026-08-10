// Package insights (Phase 9a) generates weekly productivity reports from
// existing domain events: task completions, pomodoro focus sessions (persisted
// by consuming pomodoro.session.completed), and recipe creations. Reports are
// template-rendered server-side with an optional LLM summary behind a feature
// flag (see docs/adr/011).
package insights

import (
	"time"
)

// HabitStat is a per-habit completion count for the report week.
type HabitStat struct {
	TaskID string `json:"task_id"`
	Title  string `json:"title"`
	Count  int    `json:"count"`
}

// WeeklyStats is the structured data the report renders from. It is persisted
// as report_json so the web client can render its own cards without re-parsing
// the prose.
type WeeklyStats struct {
	WeekStart         string      `json:"week_start"`
	TotalCompletions  int         `json:"total_completions"`
	CompletionsPerDay float64     `json:"completions_per_day"`
	TopHabits         []HabitStat `json:"top_habits"`
	FocusMinutes      int         `json:"focus_minutes"`
	FocusSessions     int         `json:"focus_sessions"`
	RecipesCreated    int         `json:"recipes_created"`
}

// InsightReport is a cached weekly report for a user.
type InsightReport struct {
	ID         string
	UserID     string
	WeekStart  time.Time
	Stats      WeeklyStats
	ReportText string
	LLMSummary *string
	CreatedAt  time.Time
}

// PomodoroSessionRecord is a persisted completed focus session.
type PomodoroSessionRecord struct {
	ID           string
	UserID       string
	FocusMinutes int
	ElapsedS     int
	StartedAt    time.Time
	CompletedAt  time.Time
}

// CompletionRecord is a task completion joined with its task title.
type CompletionRecord struct {
	TaskID        string
	Title         string
	CompletedDate time.Time
}
