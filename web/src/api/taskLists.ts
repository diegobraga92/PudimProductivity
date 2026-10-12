import { request } from "./client";
import type { components } from "./generated/tasks-v1";
import type { Task } from "./tasks";

// Types are generated from api/openapi/tasks-v1.yaml (the source of truth).
export type TaskList = components["schemas"]["TaskList"];
export type CreateTaskListRequest = components["schemas"]["CreateTaskListRequest"];
export type UpdateTaskListRequest = components["schemas"]["UpdateTaskListRequest"];

// Single-list endpoints answer 404 with the same message.
const NOT_FOUND = "Task list not found";

/** Fetches all task lists. */
export async function listTaskLists(): Promise<TaskList[]> {
  return request<TaskList[]>("/task-lists", { action: "list task lists" });
}

/** Creates a new task list. */
export async function createTaskList(req: CreateTaskListRequest): Promise<TaskList> {
  return request<TaskList>("/task-lists", {
    method: "POST",
    body: JSON.stringify(req),
    action: "create task list",
  });
}

/** Fetches a single task list by ID. */
export async function getTaskList(listId: string): Promise<TaskList> {
  return request<TaskList>(`/task-lists/${listId}`, {
    action: "get task list",
    notFound: NOT_FOUND,
  });
}

/** Updates an existing task list. */
export async function updateTaskList(
  listId: string,
  req: UpdateTaskListRequest,
): Promise<TaskList> {
  return request<TaskList>(`/task-lists/${listId}`, {
    method: "PUT",
    body: JSON.stringify(req),
    action: "update task list",
    notFound: NOT_FOUND,
  });
}

/** Deletes a task list by ID. */
export async function deleteTaskList(listId: string): Promise<void> {
  return request<void>(`/task-lists/${listId}`, {
    method: "DELETE",
    action: "delete task list",
    notFound: NOT_FOUND,
  });
}

/** Fetches all tasks belonging to a specific task list, optionally filtered by type. */
export async function listTasksByListID(
  listId: string,
  type?: "one-off" | "habit",
): Promise<Task[]> {
  const params = new URLSearchParams();
  if (type) params.set("type", type);
  const query = params.toString();
  return request<Task[]>(`/task-lists/${listId}/tasks${query ? `?${query}` : ""}`, {
    action: "list tasks by list",
    notFound: NOT_FOUND,
  });
}
