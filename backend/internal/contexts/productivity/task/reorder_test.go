package task

import (
	"context"
	"errors"
	"testing"
)

const (
	todoA  = "11111111-1111-4111-8111-111111111111"
	todoB  = "22222222-2222-4222-8222-222222222222"
	todoC  = "33333333-3333-4333-8333-333333333333"
	habitD = "44444444-4444-4444-8444-444444444444"
	doneE  = "55555555-5555-4555-8555-555555555555"
)

func reorderTask(t *testing.T, id string, habit bool, position int) *Task {
	t.Helper()

	var days []string
	if habit {
		days = []string{"mon"}
	}

	task, err := NewTask(id, "task "+id[:4], days)
	if err != nil {
		t.Fatalf("NewTask(%s): %v", id, err)
	}
	task.Position = position

	return task
}

func TestReorderTasks_AppliesOrder(t *testing.T) {
	a := reorderTask(t, todoA, false, 0)
	b := reorderTask(t, todoB, false, 1)
	c := reorderTask(t, todoC, false, 2)
	repo := newFakeRepo(a, b, c)
	service := NewTaskService(repo, nil, nil)

	scope := TaskScope{Status: TaskStatusTodo}
	if err := service.ReorderTasks(context.Background(), scope, []string{todoC, todoA, todoB}); err != nil {
		t.Fatalf("ReorderTasks: %v", err)
	}

	want := map[string]int{todoC: 0, todoA: 1, todoB: 2}
	for id, position := range want {
		if got := repo.store[id].Position; got != position {
			t.Errorf("position of %s: got %d, want %d", id, got, position)
		}
	}
}

func TestReorderTasks_RejectsTaskFromAnotherBucket(t *testing.T) {
	a := reorderTask(t, todoA, false, 0)
	b := reorderTask(t, todoB, false, 1)
	beatles := reorderTask(t, habitD, true, 0)
	repo := newFakeRepo(a, b, beatles)
	service := NewTaskService(repo, nil, nil)

	err := service.ReorderTasks(context.Background(), TaskScope{Status: TaskStatusTodo}, []string{todoA, habitD})
	if !errors.Is(err, ErrTaskScopeMismatch) {
		t.Fatalf("got %v, want ErrTaskScopeMismatch", err)
	}

	if a.Position != 0 || b.Position != 1 {
		t.Errorf("positions changed on a rejected reorder: a=%d b=%d", a.Position, b.Position)
	}
}

func TestReorderTasks_RejectsUnknownTask(t *testing.T) {
	repo := newFakeRepo(reorderTask(t, todoA, false, 0))
	service := NewTaskService(repo, nil, nil)

	err := service.ReorderTasks(context.Background(), TaskScope{Status: TaskStatusTodo}, []string{todoA, todoB})
	if !errors.Is(err, ErrTaskScopeMismatch) {
		t.Fatalf("got %v, want ErrTaskScopeMismatch", err)
	}
}

func TestReorderTasks_RejectsBadPayloads(t *testing.T) {
	service := NewTaskService(newFakeRepo(reorderTask(t, todoA, false, 0)), nil, nil)
	scope := TaskScope{Status: TaskStatusTodo}

	cases := map[string][]string{
		"empty":      {},
		"duplicate":  {todoA, todoA},
		"not a uuid": {"nope"},
		"mixed case": {todoA, "11111111-1111-4111-8111-11111111111Z"},
	}

	for name, ids := range cases {
		t.Run(name, func(t *testing.T) {
			err := service.ReorderTasks(context.Background(), scope, ids)
			if !errors.Is(err, ErrInvalidTaskOrder) {
				t.Fatalf("got %v, want ErrInvalidTaskOrder", err)
			}
		})
	}
}

// Moving a task to another bucket must not carry its old rank along, that rank
// only means something inside the bucket it was set in.
func TestUpdateTask_StatusChangeAppendsToDestination(t *testing.T) {
	a := reorderTask(t, todoA, false, 3)
	done := reorderTask(t, doneE, false, 5)
	done.Status = TaskStatusDone
	repo := newFakeRepo(a, done)
	service := NewTaskService(repo, nil, nil)

	status := TaskStatusDone
	updated, err := service.UpdateTask(context.Background(), todoA, nil, &status, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	if updated.Position != 6 {
		t.Errorf("Position: got %d, want 6 (last in the done bucket)", updated.Position)
	}
	if repo.store[todoA].Position != 6 {
		t.Errorf("persisted position: got %d, want 6", repo.store[todoA].Position)
	}
}

func TestTaskScope_Equal(t *testing.T) {
	listA := "770e8400-e29b-41d4-a716-446655440002"
	listB := "880e8400-e29b-41d4-a716-446655440003"

	base := TaskScope{ListID: &listA, Status: TaskStatusTodo}
	sameListOtherPointer := TaskScope{ListID: stringPtr(listA), Status: TaskStatusTodo}

	if !base.Equal(sameListOtherPointer) {
		t.Error("scopes with the same list id should be equal")
	}
	if base.Equal(TaskScope{ListID: &listB, Status: TaskStatusTodo}) {
		t.Error("scopes with different lists should differ")
	}
	if base.Equal(TaskScope{ListID: &listA, IsHabit: true, Status: TaskStatusTodo}) {
		t.Error("habit and one-off scopes should differ")
	}
	if base.Equal(TaskScope{ListID: &listA, Status: TaskStatusDone}) {
		t.Error("scopes with different statuses should differ")
	}
	if !(TaskScope{Status: TaskStatusTodo}).Equal(TaskScope{Status: TaskStatusTodo}) {
		t.Error("two inbox scopes should be equal")
	}
	if (TaskScope{Status: TaskStatusTodo}).Equal(base) {
		t.Error("the inbox is not the same bucket as a list")
	}
}

func stringPtr(s string) *string {
	return &s
}
