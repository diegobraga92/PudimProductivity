package task

import (
	"strings"
	"testing"
	"time"
)

// ── NewTask ───────────────────────────────────────────────────────────────────

func TestNewTask_ValidOneOff(t *testing.T) {
	task, err := NewTask("id-1", "Buy groceries", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.ID != "id-1" {
		t.Errorf("ID: got %q, want %q", task.ID, "id-1")
	}

	if task.Title != "Buy groceries" {
		t.Errorf("Title: got %q, want %q", task.Title, "Buy groceries")
	}

	if task.Status != TaskStatusTodo {
		t.Errorf("Status: got %q, want %q", task.Status, TaskStatusTodo)
	}

	if task.IsHabit() {
		t.Error("IsHabit: expected false for one-off task")
	}

	if task.RecurrenceDays != nil {
		t.Errorf("RecurrenceDays: got %v, want nil", task.RecurrenceDays)
	}
}

func TestNewTask_ValidHabit(t *testing.T) {
	task, err := NewTask("id-2", "Morning run", []string{"mon", "wed", "fri"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !task.IsHabit() {
		t.Error("IsHabit: expected true for habit task")
	}

	if len(task.RecurrenceDays) != 3 {
		t.Errorf("RecurrenceDays length: got %d, want 3", len(task.RecurrenceDays))
	}
}

func TestNewTask_ClonesRecurrenceDays(t *testing.T) {
	days := []string{"mon", "wed"}

	task, err := NewTask("id-1", "Exercise", days)
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}

	days[0] = "fri"

	if task.RecurrenceDays[0] != "mon" {
		t.Fatal("task recurrence days should be cloned")
	}
}

func TestNewTask_EmptyID(t *testing.T) {
	_, err := NewTask("", "Buy groceries", nil)
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
}

func TestNewTask_EmptyTitle(t *testing.T) {
	_, err := NewTask("id-1", "", nil)
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestNewTask_EmptyRecurrenceDaysSlice(t *testing.T) {
	_, err := NewTask("id-1", "Exercise", []string{})
	if err == nil {
		t.Fatal("expected error for empty recurrence days slice, got nil")
	}
}

func TestNewTask_InvalidRecurrenceDay(t *testing.T) {
	_, err := NewTask("id-1", "Exercise", []string{"mon", "xyz"})
	if err == nil {
		t.Fatal("expected error for invalid recurrence day, got nil")
	}

	if !strings.Contains(err.Error(), "xyz") {
		t.Errorf("expected error message to mention 'xyz', got: %v", err)
	}
}

func TestNewTask_DuplicateRecurrenceDay(t *testing.T) {
	_, err := NewTask("id-1", "Exercise", []string{"mon", "mon"})
	if err == nil {
		t.Fatal("expected error for duplicate recurrence day, got nil")
	}
}

// ── TaskStatus.Valid ──────────────────────────────────────────────────────────

func TestTaskStatus_Valid(t *testing.T) {
	tests := []struct {
		name   string
		status TaskStatus
		valid  bool
	}{
		{
			name:   "todo",
			status: TaskStatusTodo,
			valid:  true,
		},
		{
			name:   "done",
			status: TaskStatusDone,
			valid:  true,
		},
		{
			name:   "invalid",
			status: TaskStatus("invalid"),
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.Valid()

			if got != tt.valid {
				t.Errorf("Valid(): got %v, want %v", got, tt.valid)
			}
		})
	}
}

// ── Task.Update ───────────────────────────────────────────────────────────────

func mustNewTask(t *testing.T, title string, recurrenceDays []string) *Task {
	t.Helper()

	task, err := NewTask("id-1", title, recurrenceDays)
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}

	return task
}

func TestTask_Update_Title(t *testing.T) {
	task := mustNewTask(t, "Old title", nil)

	before := task.UpdatedAt

	newTitle := "New title"

	if err := task.Update(&newTitle, nil, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if task.Title != "New title" {
		t.Errorf("Title: got %q, want %q", task.Title, "New title")
	}

	if !task.UpdatedAt.After(before) && !task.UpdatedAt.Equal(before) {
		t.Error("UpdatedAt should not go backwards")
	}
}

func TestTask_Update_EmptyTitleIsRejected(t *testing.T) {
	task := mustNewTask(t, "Original", nil)

	empty := ""

	if err := task.Update(&empty, nil, nil, nil, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("expected error for empty title, got nil")
	}

	if task.Title != "Original" {
		t.Error("Title should not be mutated on error")
	}
}

func TestTask_Update_Status(t *testing.T) {
	task := mustNewTask(t, "Do laundry", nil)

	done := TaskStatusDone

	if err := task.Update(nil, &done, nil, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if task.Status != TaskStatusDone {
		t.Errorf("Status: got %q, want %q", task.Status, TaskStatusDone)
	}
}

func TestTask_Update_InvalidStatusIsRejected(t *testing.T) {
	task := mustNewTask(t, "Do laundry", nil)

	bad := TaskStatus("invalid")

	if err := task.Update(nil, &bad, nil, nil, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
}

func TestTask_Update_RecurrenceDays(t *testing.T) {
	task := mustNewTask(t, "Exercise", nil)

	days := []string{"mon", "fri"}

	if err := task.Update(nil, nil, &days, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if !task.IsHabit() {
		t.Fatal("expected task to become a habit")
	}

	if len(task.RecurrenceDays) != 2 {
		t.Fatalf("RecurrenceDays length: got %d, want 2", len(task.RecurrenceDays))
	}
}

func TestTask_Update_ClonesRecurrenceDays(t *testing.T) {
	task := mustNewTask(t, "Exercise", nil)

	days := []string{"mon", "wed"}

	if err := task.Update(nil, nil, &days, nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}

	days[0] = "fri"

	if task.RecurrenceDays[0] != "mon" {
		t.Fatal("task recurrence days should be cloned")
	}
}

func TestTask_Update_EmptyRecurrenceDaysRejected(t *testing.T) {
	task := mustNewTask(t, "Exercise", []string{"mon"})

	empty := []string{}

	err := task.Update(nil, nil, &empty, nil, nil, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty recurrence days")
	}

	if !task.IsHabit() {
		t.Fatal("task should remain unchanged on error")
	}
}

func TestTask_Update_InvalidRecurrenceDayIsRejected(t *testing.T) {
	task := mustNewTask(t, "Exercise", nil)

	bad := []string{"mon", "xyz"}

	if err := task.Update(nil, nil, &bad, nil, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("expected error for invalid recurrence day, got nil")
	}
}

func TestTask_Update_DuplicateRecurrenceDayIsRejected(t *testing.T) {
	task := mustNewTask(t, "Exercise", nil)

	bad := []string{"mon", "mon"}

	if err := task.Update(nil, nil, &bad, nil, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("expected error for duplicate recurrence day, got nil")
	}
}

func TestTask_Update_ListIDAssignAndUnassign(t *testing.T) {
	task := mustNewTask(t, "Do laundry", nil)

	// Assign
	listIDStr := "list-uuid"
	listIDPtr := &listIDStr

	if err := task.Update(nil, nil, nil, &listIDPtr, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Update (assign): %v", err)
	}

	if task.ListID == nil || *task.ListID != "list-uuid" {
		t.Errorf("ListID after assign: got %v, want &list-uuid", task.ListID)
	}

	// Unassign
	var nilID *string

	if err := task.Update(nil, nil, nil, &nilID, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Update (unassign): %v", err)
	}

	if task.ListID != nil {
		t.Errorf("ListID after unassign: got %v, want nil", task.ListID)
	}
}

func TestTask_Update_ListIDAbsentDoesNotChange(t *testing.T) {
	task := mustNewTask(t, "Do laundry", nil)

	listIDStr := "list-uuid"
	task.ListID = &listIDStr

	var noChange **string

	if err := task.Update(nil, nil, nil, noChange, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if task.ListID == nil || *task.ListID != "list-uuid" {
		t.Errorf("ListID should not change when listID param is nil, got %v", task.ListID)
	}
}

func TestTask_Update_AlarmMinutes(t *testing.T) {
	task := mustNewTask(t, "Stand up", nil)

	alarm := 5
	alarmPtr := &alarm

	if err := task.Update(nil, nil, nil, nil, nil, nil, nil, nil, &alarmPtr); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if task.AlarmMinutes == nil || *task.AlarmMinutes != 5 {
		t.Errorf("AlarmMinutes: got %v, want &5", task.AlarmMinutes)
	}
}

func TestTask_Update_NegativeAlarmMinutesRejected(t *testing.T) {
	task := mustNewTask(t, "Stand up", nil)

	bad := -5
	badPtr := &bad

	if err := task.Update(nil, nil, nil, nil, nil, nil, nil, nil, &badPtr); err == nil {
		t.Fatal("expected error for negative alarm_minutes, got nil")
	}
}

// ── NewTaskWithSchedule ───────────────────────────────────────────────────────

func TestNewTaskWithSchedule_ValidWithAlarm(t *testing.T) {
	start := "09:00"
	end := "10:00"
	alarm := 10

	task, err := NewTaskWithSchedule("id-1", "Morning run", []string{"mon", "wed", "fri"}, &start, &end, nil, nil, &alarm)
	if err != nil {
		t.Fatalf("NewTaskWithSchedule: %v", err)
	}

	if task.AlarmMinutes == nil || *task.AlarmMinutes != 10 {
		t.Errorf("AlarmMinutes: got %v, want &10", task.AlarmMinutes)
	}
}

func TestNewTaskWithSchedule_NegativeAlarmRejected(t *testing.T) {
	start := "09:00"
	end := "10:00"
	bad := -1

	_, err := NewTaskWithSchedule("id-1", "Morning run", []string{"mon"}, &start, &end, nil, nil, &bad)
	if err == nil {
		t.Fatal("expected error for negative alarm_minutes, got nil")
	}
}

// ── NewTaskCompletion ─────────────────────────────────────────────────────────

func TestNewTaskCompletion_Valid(t *testing.T) {
	date := time.Date(2026, 5, 25, 10, 30, 0, 0, time.UTC)

	c, err := NewTaskCompletion("cid-1", "tid-1", date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.TaskID != "tid-1" {
		t.Errorf("TaskID: got %q, want %q", c.TaskID, "tid-1")
	}

	if !c.CompletedDate.Equal(date.UTC()) {
		t.Errorf("CompletedDate: got %v, want %v", c.CompletedDate, date.UTC())
	}
}

func TestNewTaskCompletion_EmptyID(t *testing.T) {
	_, err := NewTaskCompletion("", "tid-1", time.Now())
	if err == nil {
		t.Fatal("expected error for empty completion id, got nil")
	}
}

func TestNewTaskCompletion_EmptyTaskID(t *testing.T) {
	_, err := NewTaskCompletion("cid-1", "", time.Now())
	if err == nil {
		t.Fatal("expected error for empty task id, got nil")
	}
}

func TestNewTaskCompletion_ZeroCompletedDate(t *testing.T) {
	_, err := NewTaskCompletion("cid-1", "tid-1", time.Time{})
	if err == nil {
		t.Fatal("expected error for zero completed date, got nil")
	}
}
