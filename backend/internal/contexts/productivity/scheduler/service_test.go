package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/contexts/productivity/task"
)

// fakeTasks implements TaskReader.
type fakeTasks struct {
	todos       []*task.Task
	habits      []*task.Task
	scheduled   []*task.Task
	completions []*task.TaskCompletion
}

func (f fakeTasks) ListTasks(_ context.Context, status, typ string) ([]*task.Task, error) {
	if typ == "habit" {
		return f.habits, nil
	}
	if status == "todo" {
		return f.todos, nil
	}
	return nil, nil
}
func (f fakeTasks) ListScheduledTasks(context.Context) ([]*task.Task, error) { return f.scheduled, nil }
func (f fakeTasks) GetAllTaskCompletions(context.Context, time.Time, time.Time) ([]*task.TaskCompletion, error) {
	return f.completions, nil
}

func strp(s string) *string { return &s }

func TestSuggestDay_HabitThenTodo(t *testing.T) {
	// Monday 2026-08-10.
	date := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	tasks := fakeTasks{
		todos:  []*task.Task{{ID: "t1", Title: "Buy milk"}},
		habits: []*task.Task{{ID: "h1", Title: "Workout", RecurrenceDays: []string{"mon", "wed", "fri"}}},
	}
	svc := NewSchedulerService(tasks)

	sugg, err := svc.SuggestDay(context.Background(), date)
	if err != nil {
		t.Fatalf("SuggestDay: %v", err)
	}
	if len(sugg.Slots) != 2 {
		t.Fatalf("want 2 slots, got %+v", sugg.Slots)
	}
	// Habit first (09:00–09:30), then the todo (09:30–10:00).
	if sugg.Slots[0].Kind != SlotHabit || sugg.Slots[0].Title != "Workout" || sugg.Slots[0].StartTime != "09:00" {
		t.Fatalf("slot[0] = %+v", sugg.Slots[0])
	}
	if sugg.Slots[1].Kind != SlotTodo || sugg.Slots[1].StartTime != "09:30" {
		t.Fatalf("slot[1] = %+v", sugg.Slots[1])
	}
	if sugg.FreeHours != 8 { // 9h work day - 1h of slots
		t.Fatalf("free hours = %d, want 8", sugg.FreeHours)
	}
}

func TestSuggestDay_RespectsPlannedBlock(t *testing.T) {
	date := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	tasks := fakeTasks{
		todos: []*task.Task{
			{ID: "t1", Title: "Alpha"},
			{ID: "t2", Title: "Beta"},
			{ID: "t3", Title: "Gamma"},
		},
		// Team standup blocks 10:00–11:00 on Monday.
		scheduled: []*task.Task{{ID: "s1", Title: "Standup", StartTime: strp("10:00"), EndTime: strp("11:00"), ScheduledDate: strp("2026-08-10")}},
	}
	svc := NewSchedulerService(tasks)

	sugg, err := svc.SuggestDay(context.Background(), date)
	if err != nil {
		t.Fatalf("SuggestDay: %v", err)
	}
	// Slots must not overlap the occupied 10:00–11:00 block.
	for _, s := range sugg.Slots {
		if s.StartTime >= "10:00" && s.StartTime < "11:00" {
			t.Fatalf("slot overlaps planned block: %+v", s)
		}
	}
	if len(sugg.Slots) != 3 {
		t.Fatalf("want 3 slots, got %d", len(sugg.Slots))
	}
}

func TestBuildProfile(t *testing.T) {
	// Completions only between 8am and 6pm.
	completions := []*task.TaskCompletion{
		{CreatedAt: time.Date(2026, 8, 9, 8, 30, 0, 0, time.UTC)},
		{CreatedAt: time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)},
		{CreatedAt: time.Date(2026, 8, 9, 18, 0, 0, 0, time.UTC)},
	}
	p := buildProfile(completions)
	if p.WorkStartHour != 8 {
		t.Fatalf("work start = %d, want 8", p.WorkStartHour)
	}
	if p.WorkEndHour != 19 {
		t.Fatalf("work end = %d, want 19", p.WorkEndHour)
	}
	if p.AvgPerDay != 3.0/14.0 {
		t.Fatalf("avg per day = %v", p.AvgPerDay)
	}
}

func TestSuggestDay_ExcludesAlreadyScheduled(t *testing.T) {
	date := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	tasks := fakeTasks{
		todos: []*task.Task{
			// Already time-blocked — must not be suggested again.
			{ID: "s1", Title: "Team standup", StartTime: strp("10:00"), EndTime: strp("11:00"), ScheduledDate: strp("2026-08-10")},
			{ID: "t1", Title: "Buy milk"},
		},
		scheduled: []*task.Task{{ID: "s1", Title: "Team standup", StartTime: strp("10:00"), EndTime: strp("11:00"), ScheduledDate: strp("2026-08-10")}},
	}
	svc := NewSchedulerService(tasks)

	sugg, err := svc.SuggestDay(context.Background(), date)
	if err != nil {
		t.Fatalf("SuggestDay: %v", err)
	}
	if len(sugg.Slots) != 1 {
		t.Fatalf("want exactly 1 suggested slot, got %+v", sugg.Slots)
	}
	if sugg.Slots[0].Title != "Buy milk" {
		t.Fatalf("slot = %+v, want Buy milk", sugg.Slots[0])
	}
	// Buy milk must land in a gap, not inside the 10:00-11:00 block.
	if sugg.Slots[0].StartTime >= "10:00" && sugg.Slots[0].StartTime < "11:00" {
		t.Fatalf("slot overlaps planned block: %+v", sugg.Slots[0])
	}
}

func TestFreeBlocks(t *testing.T) {
	profile := UserProfile{WorkStartHour: 9, WorkEndHour: 18}
	occupied := []timeBlock{{start: 10 * 60, end: 11 * 60}, {start: 14 * 60, end: 15 * 60}}
	free := freeBlocks(profile, occupied)
	want := []timeBlock{{9 * 60, 10 * 60}, {11 * 60, 14 * 60}, {15 * 60, 18 * 60}}
	if len(free) != len(want) {
		t.Fatalf("free = %+v, want %+v", free, want)
	}
	for i := range want {
		if free[i] != want[i] {
			t.Fatalf("free = %+v, want %+v", free, want)
		}
	}
}
