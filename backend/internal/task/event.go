package task

import (
	"time"

	"github.com/diegobraga92/pudimproductivity/backend/internal/shared"
)

// Ensure domain events implement the shared.DomainEvent interface.
var _ shared.DomainEvent = (*TaskCreated)(nil)
var _ shared.DomainEvent = (*TaskUpdated)(nil)
var _ shared.DomainEvent = (*TaskDeleted)(nil)
var _ shared.DomainEvent = (*TaskStatusChanged)(nil)

// TaskCreated is emitted when a new task is created.
type TaskCreated struct {
	Task     *Task
	occurred time.Time
}

func NewTaskCreated(task *Task) *TaskCreated {
	return &TaskCreated{
		Task:     task,
		occurred: time.Now().UTC(),
	}
}

func (e *TaskCreated) EventName() string       { return "task.created" }
func (e *TaskCreated) AggregateID() string      { return e.Task.ID }
func (e *TaskCreated) OccurredAt() time.Time    { return e.occurred }

// TaskUpdated is emitted when an existing task is updated.
type TaskUpdated struct {
	Task     *Task
	occurred time.Time
}

func NewTaskUpdated(task *Task) *TaskUpdated {
	return &TaskUpdated{
		Task:     task,
		occurred: time.Now().UTC(),
	}
}

func (e *TaskUpdated) EventName() string       { return "task.updated" }
func (e *TaskUpdated) AggregateID() string      { return e.Task.ID }
func (e *TaskUpdated) OccurredAt() time.Time    { return e.occurred }

// TaskDeleted is emitted when a task is deleted.
type TaskDeleted struct {
	TaskID   string
	occurred time.Time
}

func NewTaskDeleted(taskID string) *TaskDeleted {
	return &TaskDeleted{
		TaskID:   taskID,
		occurred: time.Now().UTC(),
	}
}

func (e *TaskDeleted) EventName() string       { return "task.deleted" }
func (e *TaskDeleted) AggregateID() string      { return e.TaskID }
func (e *TaskDeleted) OccurredAt() time.Time    { return e.occurred }

// TaskStatusChanged is emitted when a task's status changes.
type TaskStatusChanged struct {
	Task     *Task
	OldStatus TaskStatus
	NewStatus TaskStatus
	occurred  time.Time
}

func NewTaskStatusChanged(task *Task, oldStatus TaskStatus) *TaskStatusChanged {
	return &TaskStatusChanged{
		Task:      task,
		OldStatus: oldStatus,
		NewStatus: task.Status,
		occurred:  time.Now().UTC(),
	}
}

func (e *TaskStatusChanged) EventName() string    { return "task.status_changed" }
func (e *TaskStatusChanged) AggregateID() string   { return e.Task.ID }
func (e *TaskStatusChanged) OccurredAt() time.Time { return e.occurred }
