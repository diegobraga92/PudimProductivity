import config from "../config";

export type TaskStatus = "todo" | "done";

export type RecurrenceDay = "mon" | "tue" | "wed" | "thu" | "fri" | "sat" | "sun";

export interface Task {
  id: string;
  title: string;
  status: TaskStatus;
  recurrence_days?: RecurrenceDay[];
  list_id?: string | null;
  start_time?: string | null;
  end_time?: string | null;
  color?: string | null;
  scheduled_date?: string | null;
  created_at: string;
  updated_at: string;
}

export interface TaskCompletion {
  id: string;
  task_id: string;
  completed_date: string;
  created_at: string;
}

export interface CreateTaskRequest {
  title: string;
  recurrence_days?: RecurrenceDay[];
  list_id?: string | null;
  start_time?: string | null;
  end_time?: string | null;
  color?: string | null;
  scheduled_date?: string | null;
}

export interface UpdateTaskRequest {
  title?: string;
  status?: TaskStatus;
  recurrence_days?: RecurrenceDay[] | null;
  list_id?: string | null;
  start_time?: string | null;
  end_time?: string | null;
  color?: string | null;
  scheduled_date?: string | null;
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
    throw new Error(`Failed to list tasks: ${response.status}`);
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
    throw new Error(`Failed to list scheduled tasks: ${response.status}`);
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

/**
 * Completes a habit task for a specific date.
 * @param taskId - The ID of the task to complete.
 * @param date - Optional date string in YYYY-MM-DD format. Defaults to today if omitted.
 */
export async function completeTask(taskId: string, date?: string): Promise<TaskCompletion> {
  const params = date ? `?date=${encodeURIComponent(date)}` : "";
  const response = await fetch(`${config.apiBaseUrl}/tasks/${taskId}/complete${params}`, {
    method: "POST",
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task not found");
    }
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to complete task: ${response.status}`);
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
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task not found");
    }
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to uncomplete task: ${response.status}`);
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
    throw new Error(`Failed to get all completions: ${response.status}`);
  }

  return response.json() as Promise<TaskCompletion[]>;
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
    if (response.status === 404) {
      throw new Error("Task not found");
    }
    throw new Error(`Failed to get completions: ${response.status}`);
  }

  return response.json() as Promise<TaskCompletion[]>;
}