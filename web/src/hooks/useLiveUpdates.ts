import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { syncClient, type WsEvent } from "../api/sync";
import type { Task } from "../api/tasks";

const ONE_OFF_KEY = ["tasks", "one-off"] as const;
const HABIT_KEY = ["tasks", "habit"] as const;
const SCHEDULED_KEY = ["scheduledTasks"] as const;
const COMPLETIONS_KEY = ["habitCompletions"] as const;
const TASK_LISTS_KEY = ["taskLists"] as const;

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
      const key = isHabit(task) ? HABIT_KEY : ONE_OFF_KEY;
      queryClient.setQueryData<Task[]>(key, (old) => upsertTask(old, task));
      // The task may have moved between one-off/habit, so reconcile both.
      const otherKey = isHabit(task) ? ONE_OFF_KEY : HABIT_KEY;
      queryClient.setQueryData<Task[]>(otherKey, (old) =>
        old ? removeTask(old, task.id) : old
      );
      queryClient.invalidateQueries({ queryKey: SCHEDULED_KEY });
    };

    const handleDeleted = (event: WsEvent) => {
      const { id } = (event.payload ?? {}) as { id?: string };
      if (!id) return;
      queryClient.setQueryData<Task[]>(ONE_OFF_KEY, (old) => removeTask(old, id));
      queryClient.setQueryData<Task[]>(HABIT_KEY, (old) => removeTask(old, id));
      queryClient.invalidateQueries({ queryKey: SCHEDULED_KEY });
    };

    const handleCompletionChanged = () => {
      queryClient.invalidateQueries({ queryKey: COMPLETIONS_KEY });
      queryClient.invalidateQueries({ queryKey: SCHEDULED_KEY });
    };

    const handleStale = () => {
      queryClient.invalidateQueries({ queryKey: ["tasks"] });
      queryClient.invalidateQueries({ queryKey: COMPLETIONS_KEY });
      queryClient.invalidateQueries({ queryKey: SCHEDULED_KEY });
      queryClient.invalidateQueries({ queryKey: TASK_LISTS_KEY });
    };

    // A task merge resolved — the payload is the winning task, so the cache
    // converges exactly like task.updated.
    const handleMerge = (event: WsEvent) => {
      handleUpsert(event);
      queryClient.invalidateQueries({ queryKey: SCHEDULED_KEY });
    };

    // Membership changed (list shared with me / unshared). Refresh the list
    // picker and any tasks cached under that list.
    const handleShareChanged = (event: WsEvent) => {
      queryClient.invalidateQueries({ queryKey: TASK_LISTS_KEY });
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
