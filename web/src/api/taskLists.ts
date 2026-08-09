import config from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/tasks-v1";

// Types are generated from api/openapi/tasks-v1.yaml (the source of truth).
export type TaskList = components["schemas"]["TaskList"];
export type CreateTaskListRequest = components["schemas"]["CreateTaskListRequest"];
export type UpdateTaskListRequest = components["schemas"]["UpdateTaskListRequest"];

/**
 * Fetches all task lists.
 */
export async function listTaskLists(): Promise<TaskList[]> {
  const response = await fetch(`${config.apiBaseUrl}/task-lists`);
  if (!response.ok) {
    throw new Error(`Failed to list task lists: ${response.status}`);
  }
  return response.json() as Promise<TaskList[]>;
}

/**
 * Creates a new task list.
 */
export async function createTaskList(req: CreateTaskListRequest): Promise<TaskList> {
  const response = await fetch(`${config.apiBaseUrl}/task-lists`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to create task list: ${response.status}`);
  }

  return response.json() as Promise<TaskList>;
}

/**
 * Fetches a single task list by ID.
 */
export async function getTaskList(listId: string): Promise<TaskList> {
  const response = await fetch(`${config.apiBaseUrl}/task-lists/${listId}`);

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task list not found");
    }
    throw new Error(`Failed to get task list: ${response.status}`);
  }

  return response.json() as Promise<TaskList>;
}

/**
 * Updates an existing task list.
 */
export async function updateTaskList(
  listId: string,
  req: UpdateTaskListRequest
): Promise<TaskList> {
  const response = await fetch(`${config.apiBaseUrl}/task-lists/${listId}`, {
    method: "PUT",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task list not found");
    }
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to update task list: ${response.status}`);
  }

  return response.json() as Promise<TaskList>;
}

/**
 * Deletes a task list by ID.
 */
export async function deleteTaskList(listId: string): Promise<void> {
  const response = await fetch(`${config.apiBaseUrl}/task-lists/${listId}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task list not found");
    }
    throw new Error(`Failed to delete task list: ${response.status}`);
  }
}

/**
 * Fetches all tasks belonging to a specific task list, optionally filtered by type.
 */
export async function listTasksByListID(listId: string, type?: "one-off" | "habit"): Promise<import("./tasks").Task[]> {
  const params = new URLSearchParams();
  if (type) params.set("type", type);

  const query = params.toString();
  const url = `${config.apiBaseUrl}/task-lists/${listId}/tasks${query ? `?${query}` : ""}`;
  const response = await fetch(url);

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task list not found");
    }
    throw new Error(`Failed to list tasks by list: ${response.status}`);
  }

  return response.json() as Promise<import("./tasks").Task[]>;
}
