import config from "../config";

export interface TaskList {
  id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTaskListRequest {
  name: string;
}

export interface UpdateTaskListRequest {
  name?: string;
  description?: string;
}

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
    headers: { "Content-Type": "application/json" },
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
    headers: { "Content-Type": "application/json" },
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
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task list not found");
    }
    throw new Error(`Failed to delete task list: ${response.status}`);
  }
}

/**
 * Fetches all tasks belonging to a specific task list.
 */
export async function listTasksByListID(listId: string): Promise<import("./tasks").Task[]> {
  const response = await fetch(`${config.apiBaseUrl}/task-lists/${listId}/tasks`);

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task list not found");
    }
    throw new Error(`Failed to list tasks by list: ${response.status}`);
  }

  return response.json() as Promise<import("./tasks").Task[]>;
}
