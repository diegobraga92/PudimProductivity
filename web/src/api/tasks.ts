import config from "../config";

export type TaskStatus = "todo" | "in_progress" | "done";
export type TaskPriority = "low" | "medium" | "high";

export interface Task {
  id: string;
  title: string;
  description: string | null;
  status: TaskStatus;
  priority: TaskPriority;
  due_date: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateTaskRequest {
  title: string;
  description?: string | null;
  priority?: TaskPriority;
  due_date?: string | null;
}

export interface UpdateTaskRequest {
  title?: string;
  description?: string | null;
  status?: TaskStatus;
  priority?: TaskPriority;
  due_date?: string | null;
}

/**
 * Fetches all tasks, optionally filtered by status and/or priority.
 */
export async function listTasks(
  status?: TaskStatus,
  priority?: TaskPriority
): Promise<Task[]> {
  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (priority) params.set("priority", priority);

  const query = params.toString();
  const url = `${config.apiBaseUrl}/tasks${query ? `?${query}` : ""}`;

  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`Failed to list tasks: ${response.status}`);
  }
  return response.json() as Promise<Task[]>;
}

/**
 * Creates a new task.
 */
export async function createTask(req: CreateTaskRequest): Promise<Task> {
  const response = await fetch(`${config.apiBaseUrl}/tasks`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to create task: ${response.status}`);
  }

  return response.json() as Promise<Task>;
}

/**
 * Fetches a single task by ID.
 */
export async function getTask(taskId: string): Promise<Task> {
  const response = await fetch(`${config.apiBaseUrl}/tasks/${taskId}`);

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task not found");
    }
    throw new Error(`Failed to get task: ${response.status}`);
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
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task not found");
    }
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to update task: ${response.status}`);
  }

  return response.json() as Promise<Task>;
}

/**
 * Deletes a task by ID.
 */
export async function deleteTask(taskId: string): Promise<void> {
  const response = await fetch(`${config.apiBaseUrl}/tasks/${taskId}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task not found");
    }
    throw new Error(`Failed to delete task: ${response.status}`);
  }
}
