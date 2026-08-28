import config from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/tasks-v1";

// Types are generated from api/openapi/tasks-v1.yaml (the source of truth).
export type Task = components["schemas"]["Task"];
export type TaskStatus = components["schemas"]["TaskStatus"];
export type RecurrenceDay = components["schemas"]["RecurrenceDay"];
export type TaskCompletion = components["schemas"]["TaskCompletion"];
export type CreateTaskRequest = components["schemas"]["CreateTaskRequest"];
export type UpdateTaskRequest = components["schemas"]["UpdateTaskRequest"];

/** Parse an error response body, falling back to a generic message. */
async function parseError(response: Response, fallback: string): Promise<Error> {
  try {
    const body = await response.json();
    return new Error(body.error || fallback);
  } catch {
    return new Error(fallback);
  }
}

/** Handle non-OK responses consistently, with 404 special-casing. */
async function handleApiError(response: Response, action: string): Promise<never> {
  if (response.status === 404) {
    throw new Error("Task not found");
  }
  throw await parseError(response, `Failed to ${action}: ${response.status}`);
}

/**
 * Fetches all tasks, optionally filtered by status and/or type.
 */
export async function listTasks(status?: TaskStatus, type?: "one-off" | "habit"): Promise<Task[]> {
  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (type) params.set("type", type);

  const query = params.toString();
  const url = `${config.apiBaseUrl}/tasks${query ? `?${query}` : ""}`;

  const response = await fetch(url);
  if (!response.ok) {
    await handleApiError(response, "list tasks");
  }
  return response.json() as Promise<Task[]>;
}

/**
 * Fetches all tasks that have scheduling info (for the Planner view).
 */
export async function listScheduledTasks(): Promise<Task[]> {
  const url = `${config.apiBaseUrl}/tasks/scheduled`;
  const response = await fetch(url);
  if (!response.ok) {
    await handleApiError(response, "list scheduled tasks");
  }
  return response.json() as Promise<Task[]>;
}

/**
 * Creates a new task.
 */
export async function createTask(req: CreateTaskRequest): Promise<Task> {
  const response = await fetch(`${config.apiBaseUrl}/tasks`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    await handleApiError(response, "create task");
  }

  return response.json() as Promise<Task>;
}

/**
 * Fetches a single task by ID.
 */
export async function getTask(taskId: string): Promise<Task> {
  const response = await fetch(`${config.apiBaseUrl}/tasks/${taskId}`);

  if (!response.ok) {
    await handleApiError(response, "get task");
  }

  return response.json() as Promise<Task>;
}

/**
 * Updates an existing task.
 */
export async function updateTask(
  taskId: string,
  req: UpdateTaskRequest
): Promise<Task> {
  const response = await fetch(`${config.apiBaseUrl}/tasks/${taskId}`, {
    method: "PUT",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    await handleApiError(response, "update task");
  }

  return response.json() as Promise<Task>;
}

/**
 * Deletes a task by ID.
 */
export async function deleteTask(taskId: string): Promise<void> {
  const response = await fetch(`${config.apiBaseUrl}/tasks/${taskId}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });

  if (!response.ok) {
    await handleApiError(response, "delete task");
  }
}

/**
 * Completes a habit task for a specific date.
 * @param taskId - The ID of the task to complete.
 * @param date - Optional date string in YYYY-MM-DD format. Defaults to today if omitted.
 */
export async function completeTask(taskId: string, date?: string): Promise<TaskCompletion> {
  const params = date ? `?date=${encodeURIComponent(date)}` : "";
  const response = await fetch(`${config.apiBaseUrl}/tasks/${taskId}/complete${params}`, {
    method: "POST",
    headers: apiHeaders(),
  });

  if (!response.ok) {
    await handleApiError(response, "complete task");
  }

  return response.json() as Promise<TaskCompletion>;
}

/**
 * Uncompletes a habit task for a specific date.
 * @param taskId - The ID of the task to uncomplete.
 * @param date - Optional date string in YYYY-MM-DD format. Defaults to today if omitted.
 */
export async function uncompleteTask(taskId: string, date?: string): Promise<void> {
  const params = date ? `?date=${encodeURIComponent(date)}` : "";
  const response = await fetch(`${config.apiBaseUrl}/tasks/${taskId}/complete${params}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });

  if (!response.ok) {
    await handleApiError(response, "uncomplete task");
  }
}

/**
 * Gets all completions across every habit task within a date range (batch endpoint).
 * Use this instead of calling getTaskCompletions per task to avoid N+1 requests.
 */
export async function getAllTaskCompletions(
  from?: string,
  to?: string
): Promise<TaskCompletion[]> {
  const params = new URLSearchParams();
  if (from) params.set("from", from);
  if (to) params.set("to", to);

  const query = params.toString();
  const url = `${config.apiBaseUrl}/tasks/completions${query ? `?${query}` : ""}`;

  const response = await fetch(url);
  if (!response.ok) {
    await handleApiError(response, "get all completions");
  }

  return response.json() as Promise<TaskCompletion[]>;
}

/** Parse a natural-language task input into structured fields. */
export async function parseTask(input: string): Promise<ParseTaskResult> {
  const response = await fetch(`${config.apiBaseUrl}/tasks/parse`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify({ input }),
  });

  if (!response.ok) {
    await handleApiError(response, "parse task");
  }

  return response.json() as Promise<ParseTaskResult>;
}

export interface ParseTaskResult {
  title?: string;
  due_date?: string | null;
  start_time?: string | null;
  end_time?: string | null;
  duration_minutes?: number;
  recurrence_days?: RecurrenceDay[];
}

/**
 * Gets completions for a habit task within a date range.
 */
export async function getTaskCompletions(
  taskId: string,
  from?: string,
  to?: string
): Promise<TaskCompletion[]> {
  const params = new URLSearchParams();
  if (from) params.set("from", from);
  if (to) params.set("to", to);

  const query = params.toString();
  const url = `${config.apiBaseUrl}/tasks/${taskId}/completions${query ? `?${query}` : ""}`;

  const response = await fetch(url);

  if (!response.ok) {
    await handleApiError(response, "get completions");
  }

  return response.json() as Promise<TaskCompletion[]>;
}