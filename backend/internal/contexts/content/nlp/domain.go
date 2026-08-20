// Package nlp implements a small rule-based parser for natural-language task
// input (Phase 7): "Buy milk tomorrow at 9am for 45 minutes" becomes a
// structured ParsedTask that the task module can pre-fill.
//
// The parser is deliberately deterministic and dependency-free. Unsupported
// patterns leave fields nil — the caller decides whether to keep the free
// text as the title or ask for clarification (see ADR 008).
package nlp

import "errors"

// ParsedTask is the structured result of parsing a natural-language input.
type ParsedTask struct {
	// Title is the remaining text after date/time/duration/recurrence
	// expressions were removed.
	Title string
	// DueDate is an ISO date (YYYY-MM-DD) resolved from relative phrases
	// (today/tomorrow/next monday) or an explicit date.
	DueDate *string
	// StartTime is an HH:MM 24-hour string resolved from "at 3pm"/"14:30".
	StartTime *string
	// EndTime is computed from StartTime + DurationMinutes when both exist.
	EndTime *string
	// DurationMinutes is parsed from "for 45 min"/"1 hour 30 minutes"/"45m".
	DurationMinutes int
	// RecurrenceDays holds weekday names (mon..sun) for habit-style input
	// ("every mon,wed,fri"). Empty means a one-off task.
	RecurrenceDays []string
}

// ErrNoParse is returned when the input contains no recognizable structure at
// all (nothing could be extracted and no title remains).
var ErrNoParse = errors.New("nlp: could not parse input")
