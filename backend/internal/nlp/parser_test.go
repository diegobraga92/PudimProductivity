package nlp

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

// now is a fixed Monday (2026-08-10) so relative-date tests are deterministic.
var now = time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)

func TestParse(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   ParsedTask
		err    bool
	}{
		{
			name:  "basic with tomorrow and time",
			input: "Buy milk tomorrow at 9am",
			want:  ParsedTask{Title: "Buy milk", DueDate: strp("2026-08-11"), StartTime: strp("09:00")},
		},
		{
			name:  "recurrence and duration",
			input: "Workout every mon,wed,fri for 30 min",
			want:  ParsedTask{Title: "Workout", DurationMinutes: 30, RecurrenceDays: []string{"mon", "wed", "fri"}},
		},
		{
			name:  "next weekday with pm time",
			input: "Plan trip next friday at 2:30pm",
			want:  ParsedTask{Title: "Plan trip", DueDate: strp("2026-08-21"), StartTime: strp("14:30")},
		},
		{
			name:  "bare number is not a time",
			input: "Read 20 pages today",
			want:  ParsedTask{Title: "Read 20 pages", DueDate: strp("2026-08-10")},
		},
		{
			name:  "in N days",
			input: "Submit report in 3 days",
			want:  ParsedTask{Title: "Submit report", DueDate: strp("2026-08-13")},
		},
		{
			name:  "month day",
			input: "Call dentist on aug 20",
			want:  ParsedTask{Title: "Call dentist", DueDate: strp("2026-08-20")},
		},
		{
			name:  "time with duration computes end",
			input: "Yoga at 7am for 1 hour",
			want:  ParsedTask{Title: "Yoga", StartTime: strp("07:00"), DurationMinutes: 60, EndTime: strp("08:00")},
		},
		{
			name:  "daily keyword",
			input: "Journal every day",
			want:  ParsedTask{Title: "Journal", RecurrenceDays: []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}},
		},
		{
			name:  "title only",
			input: "Meditate",
			want:  ParsedTask{Title: "Meditate"},
		},
		{
			name:  "empty input",
			input: "",
			err:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input, now)
			if tc.err {
				if !errors.Is(err, ErrNoParse) {
					t.Fatalf("Parse(%q) err = %v, want ErrNoParse", tc.input, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected err: %v", tc.input, err)
			}
			if !reflect.DeepEqual(flatten(*got), flatten(tc.want)) {
				t.Fatalf("Parse(%q) = %+v, want %+v", tc.input, *got, tc.want)
			}
		})
	}
}

type flat struct {
	Title           string
	DueDate         string
	StartTime       string
	EndTime         string
	DurationMinutes int
	RecurrenceDays  []string
}

// flatten dereferences the optional pointers so DeepEqual compares values.
func flatten(p ParsedTask) flat {
	return flat{
		Title:           p.Title,
		DueDate:         deref(p.DueDate),
		StartTime:       deref(p.StartTime),
		EndTime:         deref(p.EndTime),
		DurationMinutes: p.DurationMinutes,
		RecurrenceDays:  p.RecurrenceDays,
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func strp(s string) *string { return &s }
