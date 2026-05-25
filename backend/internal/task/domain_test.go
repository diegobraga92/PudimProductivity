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
		t.Fatal("expected error for empty recurrence_days slice (use nil for one-off), got nil")
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

// ── Task.Update ───────────────────────────────────────────────────────────────

func mustNewTask(t *testing.T, id, title string, recurrenceDays []string) *Task {
	t.Helper()
	task, err := NewTask(id, title, recurrenceDays)
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	return task
}

func TestTask_Update_Title(t *testing.T) {
	task := mustNewTask(t, "id-1", "Old title", nil)
	before := task.UpdatedAt

	time.Sleep(time.Millisecond)
	newTitle := "New title"
	if err := task.Update(&newTitle, nil, nil, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if task.Title != "New title" {
		t.Errorf("Title: got %q, want %q", task.Title, "New title")
	}
	if !task.UpdatedAt.After(before) {
		t.Error("UpdatedAt should advance after Update")
	}
}

func TestTask_Update_EmptyTitleIsRejected(t *testing.T) {
	task := mustNewTask(t, "id-1", "Original", nil)
	empty := ""
	if err := task.Update(&empty, nil, nil, nil); err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
	if task.Title != "Original" {
		t.Error("Title should not be mutated on error")
	}
}

func TestTask_Update_Status(t *testing.T) {
	task := mustNewTask(t, "id-1", "Do laundry", nil)
	done := TaskStatusDone
	if err := task.Update(nil, &done, nil, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if task.Status != TaskStatusDone {
		t.Errorf("Status: got %q, want %q", task.Status, TaskStatusDone)
	}
}

func TestTask_Update_InvalidStatusIsRejected(t *testing.T) {
	task := mustNewTask(t, "id-1", "Do laundry", nil)
	bad := TaskStatus("invalid")
	if err := task.Update(nil, &bad, nil, nil); err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
}

func TestTask_Update_RecurrenceDaysNilSliceConvertsToOneOff(t *testing.T) {
	task := mustNewTask(t, "id-1", "Exercise", []string{"mon", "fri"})
	empty := []string{}
	if err := task.Update(nil, nil, &empty, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if task.IsHabit() {
		t.Error("expected task to become one-off after setting recurrence_days to []")
	}
}

func TestTask_Update_InvalidRecurrenceDayIsRejected(t *testing.T) {
	task := mustNewTask(t, "id-1", "Exercise", nil)
	bad := []string{"mon", "xyz"}
	if err := task.Update(nil, nil, &bad, nil); err == nil {
		t.Fatal("expected error for invalid recurrence day, got nil")
	}
}

func TestTask_Update_ListIDAssignAndUnassign(t *testing.T) {
	task := mustNewTask(t, "id-1", "Do laundry", nil)

	// Assign: outer non-nil, inner non-nil → set to "list-uuid"
	listIDStr := "list-uuid"
	listIDPtr := &listIDStr // *string pointing to the value
	if err := task.Update(nil, nil, nil, &listIDPtr); err != nil {
		t.Fatalf("Update (assign): %v", err)
	}
	if task.ListID == nil || *task.ListID != "list-uuid" {
		t.Errorf("ListID after assign: got %v, want &list-uuid", task.ListID)
	}

	// Unassign: outer non-nil, inner nil → set to nil
	var nilID *string // typed nil *string
	if err := task.Update(nil, nil, nil, &nilID); err != nil {
		t.Fatalf("Update (unassign): %v", err)
	}
	if task.ListID != nil {
		t.Errorf("ListID after unassign: got %v, want nil", task.ListID)
	}
}

func TestTask_Update_ListIDAbsentDoesNotChange(t *testing.T) {
	task := mustNewTask(t, "id-1", "Do laundry", nil)
	listIDStr := "list-uuid"
	task.ListID = &listIDStr

	// Pass nil outer pointer (**string nil) → field absent, no change
	var noChange **string // nil **string
	if err := task.Update(nil, nil, nil, noChange); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if task.ListID == nil || *task.ListID != "list-uuid" {
		t.Errorf("ListID should not change when listID param is nil, got %v", task.ListID)
	}
}

// ── NewTaskCompletion ─────────────────────────────────────────────────────────

func TestNewTaskCompletion_Valid(t *testing.T) {
	date := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	c, err := NewTaskCompletion("cid-1", "tid-1", date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.TaskID != "tid-1" {
		t.Errorf("TaskID: got %q, want %q", c.TaskID, "tid-1")
	}
	if !c.CompletedDate.Equal(date) {
		t.Errorf("CompletedDate: got %v, want %v", c.CompletedDate, date)
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
