import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { syncClient, type WsEvent } from "../api/sync";
import type { Task } from "../api/tasks";
import { invalidateTaskLists, queryKeys } from "../utils/queryKeys";

function isHabit(task: Task): boolean {
  return Boolean(task.recurrence_days && task.recurrence_days.length > 0);
}

function upsertTask(tasks: Task[] | undefined, task: Task): Task[] {
  if (!tasks) return [task];
  const exists = tasks.some((t) => t.id === task.id);
  return exists ? tasks.map((t) => (t.id === task.id ? task : t)) : [...tasks, task];
}

function removeTask(tasks: Task[] | undefined, id: string): Task[] | undefined {
  if (!tasks) return tasks;
  return tasks.filter((t) => t.id !== id);
}

/**
 * Subscribes to the backend WebSocket stream and keeps the React Query cache in
 * sync with task changes made from any client (web, mobile). Mount this once at
 * the app root; it is reference-counted, so multiple consumers are safe.
 */
export function useLiveUpdates(): void {
  const queryClient = useQueryClient();

  useEffect(() => {
    const handleUpsert = (event: WsEvent) => {
      const task = event.payload as unknown as Task;
      if (!task?.id) return;
      const key = isHabit(task) ? queryKeys.tasksHabit : queryKeys.tasksOneOff;
      queryClient.setQueryData<Task[]>(key, (old) => upsertTask(old, task));
      // The task may have moved between one-off/habit, so reconcile both.
      const otherKey = isHabit(task) ? queryKeys.tasksOneOff : queryKeys.tasksHabit;
      queryClient.setQueryData<Task[]>(otherKey, (old) =>
        old ? removeTask(old, task.id) : old
      );
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
    };

    const handleDeleted = (event: WsEvent) => {
      const { id } = (event.payload ?? {}) as { id?: string };
      if (!id) return;
      queryClient.setQueryData<Task[]>(queryKeys.tasksOneOff, (old) => removeTask(old, id));
      queryClient.setQueryData<Task[]>(queryKeys.tasksHabit, (old) => removeTask(old, id));
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
    };

    const handleCompletionChanged = () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.habitCompletions });
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
    };

    const handleStale = () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasksAll });
      queryClient.invalidateQueries({ queryKey: queryKeys.habitCompletions });
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
      invalidateTaskLists(queryClient);
    };

    // A task merge resolved. The payload is the winning task, so the cache
    // converges exactly like task.updated.
    const handleMerge = (event: WsEvent) => {
      handleUpsert(event);
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
    };

    // Membership changed (list shared with me / unshared). Refresh the list
    // picker and any tasks cached under that list.
    const handleShareChanged = (event: WsEvent) => {
      invalidateTaskLists(queryClient);
      const payload = (event.payload ?? {}) as { list_id?: string };
      if (payload.list_id) {
        queryClient.invalidateQueries({
          queryKey: ["taskLists", payload.list_id],
          predicate: (query) =>
            query.queryKey[0] === "taskLists" &&
            String(query.queryKey[1] ?? "") === payload.list_id,
        });
      }
    };

    const offs = [
      syncClient.on("task.created", handleUpsert),
      syncClient.on("task.updated", handleUpsert),
      syncClient.on("task.deleted", handleDeleted),
      syncClient.on("task.completed", handleCompletionChanged),
      syncClient.on("task.uncompleted", handleCompletionChanged),
      syncClient.on("task.merged", handleMerge),
      syncClient.on("tasklist.shared", handleShareChanged),
      syncClient.on("tasklist.unshared", handleShareChanged),
      syncClient.on("stale", handleStale),
    ];

    syncClient.connect();
    return () => {
      offs.forEach((off) => off());
      syncClient.close();
    };
  }, [queryClient]);
}
