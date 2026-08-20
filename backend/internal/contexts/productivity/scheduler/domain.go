// Package scheduler implements the Phase 7 auto-scheduler: it derives a user
// profile from task/completion history and produces a suggested daily plan
// that fits pending work into the user's free time blocks (respecting existing
// planner entries and recurring habits).
//
// The profile is a pure function of the data (computed on demand, not
// persisted) — see ADR 009 for the design rationale.
package scheduler

import (
	"context"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
)

// TaskReader is the slice of the task module the scheduler needs. Satisfied by
// *task.TaskService.
type TaskReader interface {
	ListTasks(ctx context.Context, statusFilter, typeFilter string) ([]*task.Task, error)
	ListScheduledTasks(ctx context.Context) ([]*task.Task, error)
	GetAllTaskCompletions(ctx context.Context, from, to time.Time) ([]*task.TaskCompletion, error)
}

// SlotKind distinguishes the origin of a suggested block.
type SlotKind string

const (
	SlotHabit SlotKind = "habit"
	SlotTodo  SlotKind = "todo"
)

// ScheduleSlot is one suggested time block for a task.
type ScheduleSlot struct {
	TaskID    string   `json:"task_id"`
	Title     string   `json:"title"`
	StartTime string   `json:"start_time"` // HH:MM 24-hour
	EndTime   string   `json:"end_time"`
	Kind      SlotKind `json:"kind"`
}

// Suggestion is the full daily plan for one date.
type Suggestion struct {
	Date         string         `json:"date"`
	Slots        []ScheduleSlot `json:"slots"`
	FreeHours    int            `json:"free_hours"`
	AvgPerDay    float64        `json:"avg_per_day"` // completions/day over the last 14 days
	PendingCount int            `json:"pending_count"`
}

// UserProfile describes the user's derived working rhythm.
type UserProfile struct {
	WorkStartHour int // inclusive (default 9)
	WorkEndHour   int // exclusive (default 18)
	AvgPerDay     float64
}

const defaultDurationMinutes = 30
