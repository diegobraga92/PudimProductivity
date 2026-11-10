import type { QueryClient } from "@tanstack/react-query";
import type { Task } from "../api/tasks";
import { queryKeys } from "./queryKeys";
import { applyPositions } from "./reorder";

/**
 * Mirrors WebSocket task events into the React Query cache.
 *
 * A task is served by exactly one bucket: the list it is filed under, or one of
 * the two inbox buckets (to-dos, habits). The REST inbox endpoints only return
 * `list_id IS NULL` rows, so a list item written into an inbox bucket would
 * render as a phantom row until the next refetch drops it.
 */

/** A task is a habit when it repeats on at least one weekday. */
function isHabitTask(task: Task): boolean {
  return Boolean(task.recurrence_days && task.recurrence_days.length > 0);
}

/** Bucket that owns a task, mirroring how each REST endpoint filters. */
function bucketKeyFor(task: Task): readonly unknown[] {
  if (task.list_id) return queryKeys.taskListTasks(task.list_id);
  return isHabitTask(task) ? queryKeys.tasksHabit : queryKeys.tasksOneOff;
}

function upsert(tasks: Task[] | undefined, task: Task): Task[] {
  if (!tasks) return [task];
  return tasks.some((t) => t.id === task.id)
    ? tasks.map((t) => (t.id === task.id ? task : t))
    : [...tasks, task];
}

// Returning undefined leaves the query untouched (and never creates it).
function withoutTask(tasks: Task[] | undefined, id: string): Task[] | undefined {
  if (!tasks?.some((t) => t.id === id)) return tasks;
  return tasks.filter((t) => t.id !== id);
}

/**
 * Drops a task from every bucket except the one that now owns it, so a task
 * that moved between a list and the inbox (or between to-dos and habits)
 * cannot linger where it no longer belongs.
 */
function removeFromOtherBuckets(
  queryClient: QueryClient,
  id: string,
  keep: { inbox?: "one-off" | "habit"; listId?: string }
): void {
  if (keep.inbox !== "one-off") {
    queryClient.setQueryData<Task[]>(queryKeys.tasksOneOff, (old) => withoutTask(old, id));
  }
  if (keep.inbox !== "habit") {
    queryClient.setQueryData<Task[]>(queryKeys.tasksHabit, (old) => withoutTask(old, id));
  }
  queryClient.setQueriesData<Task[]>(
    {
      queryKey: queryKeys.taskListTasksAll,
      predicate: (query) => query.queryKey[1] !== keep.listId,
    },
    (old) => withoutTask(old, id)
  );
}

/** Applies a `task.created` / `task.updated` / `task.merged` payload. */
export function applyTaskUpsert(queryClient: QueryClient, task: Task): void {
  const key = bucketKeyFor(task);
  removeFromOtherBuckets(queryClient, task.id, {
    inbox: task.list_id ? undefined : isHabitTask(task) ? "habit" : "one-off",
    listId: task.list_id ?? undefined,
  });
  // Only touch buckets the app has already loaded: fabricating one from a
  // single event would flash an incomplete (and, for the inbox, wrong) list.
  if (queryClient.getQueryData(key) === undefined) return;
  queryClient.setQueryData<Task[]>(key, (old) => upsert(old, task));
}

/** Applies a `task.deleted` payload (which only carries the id). */
export function applyTaskDelete(queryClient: QueryClient, id: string): void {
  removeFromOtherBuckets(queryClient, id, {});
}

export interface TaskReorderPayload {
  list_id?: string | null;
  is_habit?: boolean;
  ordered_ids?: string[];
}

/** Applies a `task.reordered` payload to the bucket the drag happened in. */
export function applyTaskReorder(queryClient: QueryClient, payload: TaskReorderPayload): void {
  const orderedIds = payload.ordered_ids;
  if (!orderedIds?.length) return;
  const key = payload.list_id
    ? queryKeys.taskListTasks(payload.list_id)
    : payload.is_habit
      ? queryKeys.tasksHabit
      : queryKeys.tasksOneOff;
  queryClient.setQueryData<Task[]>(key, (old) => (old ? applyPositions(old, orderedIds) : old));
}
