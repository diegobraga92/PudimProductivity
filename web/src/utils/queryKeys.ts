import type { QueryClient } from "@tanstack/react-query";

/** Central React Query keys, so readers and invalidators cannot drift apart. */
export const queryKeys = {
  tasksOneOff: ["tasks", "one-off"] as const,
  tasksHabit: ["tasks", "habit"] as const,
  tasksAll: ["tasks"] as const,
  scheduledTasks: ["scheduledTasks"] as const,
  habitCompletions: ["habitCompletions"] as const,
  taskLists: ["taskLists"] as const,
  taskList: (listId: string) => ["taskList", listId] as const,
  /** Tasks filed under one list; the REST counterpart of the inbox buckets. */
  taskListTasks: (listId: string) => ["taskListTasks", listId] as const,
  /** Prefix matching every cached list bucket, for writes that span lists. */
  taskListTasksAll: ["taskListTasks"] as const,
  taskListMembers: (listId: string) => ["taskListMembers", listId] as const,
  pomodoroCurrent: ["pomodoro", "current"] as const,
} as const;

/** Invalidate both task buckets (one-off and habit) after a task write. */
export function invalidateTasks(queryClient: QueryClient): void {
  queryClient.invalidateQueries({ queryKey: queryKeys.tasksOneOff });
  queryClient.invalidateQueries({ queryKey: queryKeys.tasksHabit });
}

/** Invalidate the task-list picker after a list write or membership change. */
export function invalidateTaskLists(queryClient: QueryClient): void {
  queryClient.invalidateQueries({ queryKey: queryKeys.taskLists });
}
