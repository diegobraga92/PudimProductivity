package planner

import (
	"fmt"
	"time"
)

type PlannerEntry struct {
	ID        string
	Title     string
	Days      []string // ["mon", "tue", ..., "sun"]
	StartTime string   // HH:MM (24h)
	EndTime   string   // HH:MM (24h)
	Color     string   // hex color e.g. "#3B82F6"
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewPlannerEntry(id, title string, days []string, startTime, endTime, color string) (*PlannerEntry, error) {
	if id == "" {
		return nil, fmt.Errorf("planner entry id cannot be empty")
	}
	if title == "" {
		return nil, fmt.Errorf("planner entry title cannot be empty")
	}
	if err := validateDays(days); err != nil {
		return nil, err
	}
	if err := validateTime(startTime); err != nil {
		return nil, fmt.Errorf("start_time: %w", err)
	}
	if err := validateTime(endTime); err != nil {
		return nil, fmt.Errorf("end_time: %w", err)
	}
	if startTime >= endTime {
		return nil, fmt.Errorf("start_time must be before end_time")
	}
	if color == "" {
		color = "#3B82F6" // default blue
	}

	now := time.Now().UTC()
	return &PlannerEntry{
		ID:        id,
		Title:     title,
		Days:      cloneStrings(days),
		StartTime: startTime,
		EndTime:   endTime,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (e *PlannerEntry) Update(title *string, days *[]string, startTime, endTime, color *string) error {
	if title != nil {
		if *title == "" {
			return fmt.Errorf("planner entry title cannot be empty")
		}
		e.Title = *title
	}
	if days != nil {
		if err := validateDays(*days); err != nil {
			return err
		}
		e.Days = cloneStrings(*days)
	}
	if startTime != nil {
		if err := validateTime(*startTime); err != nil {
			return fmt.Errorf("start_time: %w", err)
		}
		e.StartTime = *startTime
	}
	if endTime != nil {
		if err := validateTime(*endTime); err != nil {
			return fmt.Errorf("end_time: %w", err)
		}
		e.EndTime = *endTime
	}
	if color != nil {
		if *color == "" {
			return fmt.Errorf("color cannot be empty")
		}
		e.Color = *color
	}
	e.UpdatedAt = time.Now().UTC()
	return nil
}

var validDays = map[string]struct{}{
	"mon": {}, "tue": {}, "wed": {}, "thu": {},
	"fri": {}, "sat": {}, "sun": {},
}

func validateDays(days []string) error {
	if len(days) == 0 {
		return fmt.Errorf("days cannot be empty")
	}
	seen := make(map[string]struct{}, len(days))
	for _, d := range days {
		if _, ok := validDays[d]; !ok {
			return fmt.Errorf("invalid day: %s", d)
		}
		if _, exists := seen[d]; exists {
			return fmt.Errorf("duplicate day: %s", d)
		}
		seen[d] = struct{}{}
	}
	return nil
}

func validateTime(t string) error {
	_, err := time.Parse("15:04", t)
	if err != nil {
		return fmt.Errorf("invalid time format %q, expected HH:MM (24h): %w", t, err)
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
