package task

import (
	"fmt"
	"regexp"
	"time"
)

type TaskStatus string

const (
	TaskStatusTodo TaskStatus = "todo"
	TaskStatusDone TaskStatus = "done"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case TaskStatusTodo, TaskStatusDone:
		return true
	default:
		return false
	}
}

var validRecurrenceDays = map[string]struct{}{
	"mon": {},
	"tue": {},
	"wed": {},
	"thu": {},
	"fri": {},
	"sat": {},
	"sun": {},
}

type Task struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Status         TaskStatus `json:"status"`
	RecurrenceDays []string   `json:"recurrence_days,omitempty"` // nil = one-off task, non-nil = habit
	ListID         *string    `json:"list_id,omitempty"`         // nil = not part of a list, non-nil = belongs to a task list
	StartTime      *string    `json:"start_time,omitempty"`      // nil = not scheduled on planner, "09:00" format
	EndTime        *string    `json:"end_time,omitempty"`        // nil = not scheduled on planner, "10:00" format
	Color          *string    `json:"color,omitempty"`           // nil = default (#3B82F6)
	ScheduledDate  *string    `json:"scheduled_date,omitempty"`  // nil for habits, "2026-07-20" for one-off tasks
	AlarmMinutes   *int       `json:"alarm_minutes,omitempty"`   // nil = no alarm, e.g. 5 = notify 5 min before start_time
	UpdatedBy      *string    `json:"updated_by,omitempty"`      // Phase 8: last writer, used to break LWW merge ties (nil = never merged)
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func NewTask(id, title string, recurrenceDays []string) (*Task, error) {
	if id == "" {
		return nil, fmt.Errorf("task id cannot be empty")
	}

	if title == "" {
		return nil, fmt.Errorf("task title cannot be empty")
	}

	if err := validateRecurrenceDays(recurrenceDays); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &Task{
		ID:             id,
		Title:          title,
		Status:         TaskStatusTodo,
		RecurrenceDays: cloneStrings(recurrenceDays),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func NewTaskWithSchedule(id, title string, recurrenceDays []string, startTime, endTime, color, scheduledDate *string, alarmMinutes *int) (*Task, error) {
	task, err := NewTask(id, title, recurrenceDays)
	if err != nil {
		return nil, err
	}

	if err := validateSchedule(startTime, endTime, color, scheduledDate, recurrenceDays); err != nil {
		return nil, err
	}

	if err := validateAlarmMinutes(alarmMinutes); err != nil {
		return nil, err
	}

	task.StartTime = startTime
	task.EndTime = endTime
	task.Color = color
	task.ScheduledDate = scheduledDate
	task.AlarmMinutes = alarmMinutes

	return task, nil
}

func (t *Task) IsHabit() bool {
	return t.RecurrenceDays != nil
}

func (t *Task) HasSchedule() bool {
	return t.StartTime != nil && t.EndTime != nil
}

func (t *Task) EffectiveDays() []string {
	if t.IsHabit() {
		return t.RecurrenceDays
	}
	if t.ScheduledDate != nil {
		parsed, err := time.Parse("2006-01-02", *t.ScheduledDate)
		if err != nil {
			return nil
		}
		return []string{parsed.UTC().Weekday().String()[:3]}
	}
	return nil
}

func (t *Task) Update(
	title *string,
	status *TaskStatus,
	recurrenceDays *[]string,
	listID **string,
	startTime **string,
	endTime **string,
	color **string,
	scheduledDate **string,
	alarmMinutes **int,
) error {
	if title != nil {
		if *title == "" {
			return fmt.Errorf("task title cannot be empty")
		}

		t.Title = *title
	}

	if status != nil {
		if !status.Valid() {
			return fmt.Errorf("invalid task status: %s", *status)
		}

		t.Status = *status
	}

	if recurrenceDays != nil {
		if err := validateRecurrenceDays(*recurrenceDays); err != nil {
			return err
		}

		t.RecurrenceDays = cloneStrings(*recurrenceDays)
	}

	if listID != nil {
		t.ListID = *listID
	}

	// Determine the actual start/end/color/scheduledDate values after update
	var curStartTime, curEndTime, curColor, curScheduledDate *string

	if startTime != nil {
		t.StartTime = *startTime
	}
	curStartTime = t.StartTime

	if endTime != nil {
		t.EndTime = *endTime
	}
	curEndTime = t.EndTime

	if color != nil {
		t.Color = *color
	}
	curColor = t.Color

	if scheduledDate != nil {
		t.ScheduledDate = *scheduledDate
	}
	curScheduledDate = t.ScheduledDate

	if alarmMinutes != nil {
		if err := validateAlarmMinutes(*alarmMinutes); err != nil {
			return err
		}
		t.AlarmMinutes = *alarmMinutes
	}

	// Validate schedule constraints after applying changes
	recurrenceAfter := t.RecurrenceDays
	if err := validateSchedule(curStartTime, curEndTime, curColor, curScheduledDate, recurrenceAfter); err != nil {
		return err
	}

	t.UpdatedAt = time.Now().UTC()

	return nil
}

type TaskCompletion struct {
	ID            string
	TaskID        string
	CompletedDate time.Time
	CreatedAt     time.Time
}

func NewTaskCompletion(
	id, taskID string,
	completedDate time.Time,
) (*TaskCompletion, error) {
	if id == "" {
		return nil, fmt.Errorf("completion id cannot be empty")
	}

	if taskID == "" {
		return nil, fmt.Errorf("task id cannot be empty")
	}

	if completedDate.IsZero() {
		return nil, fmt.Errorf("completed date cannot be zero")
	}

	return &TaskCompletion{
		ID:            id,
		TaskID:        taskID,
		CompletedDate: completedDate.UTC(),
		CreatedAt:     time.Now().UTC(),
	}, nil
}

func validateRecurrenceDays(days []string) error {
	if days == nil {
		return nil
	}

	if len(days) == 0 {
		return fmt.Errorf("recurrence days cannot be empty; use nil for one-off tasks")
	}

	seen := make(map[string]struct{}, len(days))

	for _, d := range days {
		if _, ok := validRecurrenceDays[d]; !ok {
			return fmt.Errorf("invalid recurrence day: %s", d)
		}

		if _, exists := seen[d]; exists {
			return fmt.Errorf("duplicate recurrence day: %s", d)
		}

		seen[d] = struct{}{}
	}

	return nil
}

func validateAlarmMinutes(alarmMinutes *int) error {
	if alarmMinutes == nil {
		return nil
	}
	if *alarmMinutes < 0 {
		return fmt.Errorf("alarm_minutes cannot be negative")
	}
	return nil
}

func validateSchedule(startTime, endTime, color, scheduledDate *string, recurrenceDays []string) error {
	if startTime == nil && endTime == nil {
		return nil
	}

	if startTime == nil {
		return fmt.Errorf("end_time requires start_time")
	}
	if endTime == nil {
		return fmt.Errorf("start_time requires end_time")
	}

	// Validate time format
	startParsed, err := time.Parse("15:04", *startTime)
	if err != nil {
		return fmt.Errorf("start_time: invalid time format %q, expected HH:MM (24h)", *startTime)
	}

	endParsed, err := time.Parse("15:04", *endTime)
	if err != nil {
		return fmt.Errorf("end_time: invalid time format %q, expected HH:MM (24h)", *endTime)
	}

	if !startParsed.Before(endParsed) {
		return fmt.Errorf("start_time must be before end_time")
	}

	// Validate color format if set
	if color != nil {
		matched, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}$`, *color)
		if !matched {
			return fmt.Errorf("color: invalid hex color format %q, expected e.g. #3B82F6", *color)
		}
	}

	// ScheduledDate is required for one-off tasks (no recurrence), forbidden for habits
	if recurrenceDays == nil && scheduledDate == nil {
		return fmt.Errorf("scheduled_date is required for one-off tasks when start_time is set")
	}
	if recurrenceDays != nil && scheduledDate != nil {
		return fmt.Errorf("scheduled_date must not be set for habits (use recurrence_days instead)")
	}

	// Validate scheduled_date format if present
	if scheduledDate != nil {
		if _, err := time.Parse("2006-01-02", *scheduledDate); err != nil {
			return fmt.Errorf("scheduled_date: invalid date format %q, expected YYYY-MM-DD", *scheduledDate)
		}
	}

	return nil
}

func cloneStrings(v []string) []string {
	if v == nil {
		return nil
	}

	cp := make([]string, len(v))
	copy(cp, v)

	return cp
}
