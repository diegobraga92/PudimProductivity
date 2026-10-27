import { request } from "./client";
import type { components } from "./generated/tasks-v1";

// Types are generated from api/openapi/tasks-v1.yaml (the source of truth).
export type Task = components["schemas"]["Task"];
export type TaskStatus = components["schemas"]["TaskStatus"];
export type RecurrenceDay = components["schemas"]["RecurrenceDay"];
export type TaskCompletion = components["schemas"]["TaskCompletion"];
export type CreateTaskRequest = components["schemas"]["CreateTaskRequest"];
export type UpdateTaskRequest = components["schemas"]["UpdateTaskRequest"];
export type ReorderTasksRequest = components["schemas"]["ReorderTasksRequest"];

// Every task endpoint answers 404 with the same message.
const NOT_FOUND = "Task not found";

/** Fetches all tasks, optionally filtered by status and/or type. */
export async function listTasks(status?: TaskStatus, type?: "one-off" | "habit"): Promise<Task[]> {
  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (type) params.set("type", type);
  const query = params.toString();
  return request<Task[]>(`/tasks${query ? `?${query}` : ""}`, {
    action: "list tasks",
    notFound: NOT_FOUND,
  });
}

/** Fetches all tasks that have scheduling info (for the Planner view). */
export async function listScheduledTasks(): Promise<Task[]> {
  return request<Task[]>("/tasks/scheduled", {
    action: "list scheduled tasks",
    notFound: NOT_FOUND,
  });
}

/** Creates a new task. */
export async function createTask(req: CreateTaskRequest): Promise<Task> {
  return request<Task>("/tasks", {
    method: "POST",
    body: JSON.stringify(req),
    action: "create task",
    notFound: NOT_FOUND,
  });
}

/** Fetches a single task by ID. */
export async function getTask(taskId: string): Promise<Task> {
  return request<Task>(`/tasks/${taskId}`, { action: "get task", notFound: NOT_FOUND });
}

/** Updates an existing task. */
export async function updateTask(taskId: string, req: UpdateTaskRequest): Promise<Task> {
  return request<Task>(`/tasks/${taskId}`, {
    method: "PUT",
    body: JSON.stringify(req),
    action: "update task",
    notFound: NOT_FOUND,
  });
}

/** Deletes a task by ID. */
export async function deleteTask(taskId: string): Promise<void> {
  return request<void>(`/tasks/${taskId}`, {
    method: "DELETE",
    action: "delete task",
    notFound: NOT_FOUND,
  });
}

/** Persists the manual order of one bucket (inbox or list, to-dos or habits). */
export async function reorderTasks(req: ReorderTasksRequest): Promise<void> {
  return request<void>("/tasks/reorder", {
    method: "PUT",
    body: JSON.stringify(req),
    action: "reorder tasks",
  });
}

/** Completes a habit for a date (YYYY-MM-DD, defaults to today). */
export async function completeTask(taskId: string, date?: string): Promise<TaskCompletion> {
  const params = date ? `?date=${encodeURIComponent(date)}` : "";
  return request<TaskCompletion>(`/tasks/${taskId}/complete${params}`, {
    method: "POST",
    action: "complete task",
    notFound: NOT_FOUND,
  });
}

/** Uncompletes a habit for a date (YYYY-MM-DD, defaults to today). */
export async function uncompleteTask(taskId: string, date?: string): Promise<void> {
  const params = date ? `?date=${encodeURIComponent(date)}` : "";
  return request<void>(`/tasks/${taskId}/complete${params}`, {
    method: "DELETE",
    action: "uncomplete task",
    notFound: NOT_FOUND,
  });
}

/** All habit completions in a date range - use instead of getTaskCompletions per task (N+1). */
export async function getAllTaskCompletions(from?: string, to?: string): Promise<TaskCompletion[]> {
  const params = new URLSearchParams();
  if (from) params.set("from", from);
  if (to) params.set("to", to);
  const query = params.toString();
  return request<TaskCompletion[]>(`/tasks/completions${query ? `?${query}` : ""}`, {
    action: "get all completions",
    notFound: NOT_FOUND,
  });
}

/** Parse a natural-language task input into structured fields. */
export async function parseTask(input: string): Promise<ParseTaskResult> {
  return request<ParseTaskResult>("/tasks/parse", {
    method: "POST",
    body: JSON.stringify({ input }),
    action: "parse task",
    notFound: NOT_FOUND,
  });
}

export interface ParseTaskResult {
  title?: string;
  due_date?: string | null;
  start_time?: string | null;
  end_time?: string | null;
  duration_minutes?: number;
  recurrence_days?: RecurrenceDay[];
}

/** Gets completions for a habit task within a date range. */
export async function getTaskCompletions(
  taskId: string,
  from?: string,
  to?: string,
): Promise<TaskCompletion[]> {
  const params = new URLSearchParams();
  if (from) params.set("from", from);
  if (to) params.set("to", to);
  const query = params.toString();
  return request<TaskCompletion[]>(`/tasks/${taskId}/completions${query ? `?${query}` : ""}`, {
    action: "get completions",
    notFound: NOT_FOUND,
  });
}
