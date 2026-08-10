package scheduler

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/task"
)

// Service derives the profile and builds daily suggestions from the task
// module's data. Stateless — safe for concurrent use.
type SchedulerService struct {
	tasks TaskReader
}

func NewSchedulerService(tasks TaskReader) *SchedulerService {
	return &SchedulerService{tasks: tasks}
}

// SuggestDay produces the suggested plan for the given date.
func (s *SchedulerService) SuggestDay(ctx context.Context, date time.Time) (*Suggestion, error) {
	weekday := weekDayKey(date.Weekday())

	// Pending one-off todos + habits.
	todos, err := s.tasks.ListTasks(ctx, "todo", "one-off")
	if err != nil {
		return nil, fmt.Errorf("scheduler: load todos: %w", err)
	}
	habits, err := s.tasks.ListTasks(ctx, "", "habit")
	if err != nil {
		return nil, fmt.Errorf("scheduler: load habits: %w", err)
	}
	scheduled, err := s.tasks.ListScheduledTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("scheduler: load scheduled: %w", err)
	}

	// Completion history for the profile (last 14 days).
	now := time.Now()
	completions, err := s.tasks.GetAllTaskCompletions(ctx, now.AddDate(0, 0, -14), now)
	if err != nil {
		return nil, fmt.Errorf("scheduler: load completions: %w", err)
	}
	profile := buildProfile(completions)

	occupied := plannedBlocks(scheduled, date, weekday)
	free := freeBlocks(profile, occupied)

	var slots []ScheduleSlot
	for _, t := range habitsForWeekday(habits, weekday) {
		start, end, ok := takeBlock(&free, defaultDurationMinutes)
		if !ok {
			break
		}
		slots = append(slots, ScheduleSlot{TaskID: t.ID, Title: t.Title, Kind: SlotHabit, StartTime: start, EndTime: end})
	}
	for _, t := range todosForDate(todos, date) {
		start, end, ok := takeBlock(&free, defaultDurationMinutes)
		if !ok {
			break
		}
		slots = append(slots, ScheduleSlot{TaskID: t.ID, Title: t.Title, Kind: SlotTodo, StartTime: start, EndTime: end})
	}

	freeHours := 0
	for _, b := range free {
		freeHours += (b.end - b.start) / 60
	}

	return &Suggestion{
		Date:         date.Format("2006-01-02"),
		Slots:        slots,
		FreeHours:    freeHours,
		AvgPerDay:    profile.AvgPerDay,
		PendingCount: len(todosForDate(todos, date)),
	}, nil
}

// buildProfile derives work hours + productivity from completion history.
func buildProfile(completions []*task.TaskCompletion) UserProfile {
	profile := UserProfile{WorkStartHour: 9, WorkEndHour: 18}
	if len(completions) == 0 {
		return profile
	}
	earliest, latest := 24, 0
	for _, c := range completions {
		h := c.CreatedAt.Hour()
		if h < earliest {
			earliest = h
		}
		if h > latest {
			latest = h
		}
	}
	if earliest >= 7 && earliest < 12 {
		profile.WorkStartHour = earliest
	}
	if latest > 12 && latest <= 21 {
		profile.WorkEndHour = latest + 1
	}
	profile.AvgPerDay = float64(len(completions)) / 14.0
	return profile
}

// timeBlock is an inclusive-start, exclusive-end minute-of-day interval.
type timeBlock struct {
	start, end int
}

// plannedBlocks collects planner entries that apply on the date.
func plannedBlocks(scheduled []*task.Task, date time.Time, weekday string) []timeBlock {
	var out []timeBlock
	for _, t := range scheduled {
		if !appliesOn(t, date, weekday) {
			continue
		}
		if t.StartTime == nil || t.EndTime == nil {
			continue
		}
		s, err1 := parseHHMM(*t.StartTime)
		e, err2 := parseHHMM(*t.EndTime)
		if err1 != nil || err2 != nil || e <= s {
			continue
		}
		out = append(out, timeBlock{s, e})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].start < out[j].start })
	return out
}

// freeBlocks returns the gaps in [workStart, workEnd) not covered by occupied.
func freeBlocks(profile UserProfile, occupied []timeBlock) []timeBlock {
	work := timeBlock{profile.WorkStartHour * 60, profile.WorkEndHour * 60}
	var out []timeBlock
	cursor := work.start
	for _, occ := range occupied {
		if occ.end <= cursor || occ.start >= work.end {
			continue
		}
		if occ.start > cursor {
			out = append(out, timeBlock{cursor, min(occ.start, work.end)})
		}
		cursor = max(cursor, occ.end)
		if cursor >= work.end {
			break
		}
	}
	if cursor < work.end {
		out = append(out, timeBlock{cursor, work.end})
	}
	return out
}

// takeBlock carves a block of the requested minutes out of the first free
// block large enough. Returns "" if none remains.
func takeBlock(free *[]timeBlock, minutes int) (string, string, bool) {
	for i, b := range *free {
		if b.end-b.start >= minutes {
			(*free)[i] = timeBlock{b.start + minutes, b.end}
			if (*free)[i].start >= (*free)[i].end {
				*free = append((*free)[:i], (*free)[i+1:]...)
			}
			return fmt.Sprintf("%02d:%02d", (b.start/60)%24, b.start%60),
				fmt.Sprintf("%02d:%02d", ((b.start+minutes)/60)%24, (b.start+minutes)%60), true
		}
	}
	return "", "", false
}

// --- task helpers ---

func habitsForWeekday(habits []*task.Task, weekday string) []*task.Task {
	var out []*task.Task
	for _, t := range habits {
		if contains(t.RecurrenceDays, weekday) {
			out = append(out, t)
		}
	}
	return out
}

func todosForDate(todos []*task.Task, date time.Time) []*task.Task {
	day := date.Format("2006-01-02")
	var out []*task.Task
	for _, t := range todos {
		// Already time-blocked tasks are handled via plannedBlocks — don't
		// suggest them again.
		if t.StartTime != nil {
			continue
		}
		// Task scheduled for this date, or unscheduled (any day).
		if t.ScheduledDate == nil || *t.ScheduledDate == day {
			out = append(out, t)
		}
	}
	return out
}

func appliesOn(t *task.Task, date time.Time, weekday string) bool {
	if t.RecurrenceDays != nil {
		return contains(t.RecurrenceDays, weekday)
	}
	return t.ScheduledDate != nil && *t.ScheduledDate == date.Format("2006-01-02")
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func weekDayKey(d time.Weekday) string {
	switch d {
	case time.Monday:
		return "mon"
	case time.Tuesday:
		return "tue"
	case time.Wednesday:
		return "wed"
	case time.Thursday:
		return "thu"
	case time.Friday:
		return "fri"
	case time.Saturday:
		return "sat"
	case time.Sunday:
		return "sun"
	}
	return ""
}

func parseHHMM(s string) (int, error) {
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, err
	}
	return h*60 + m, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
