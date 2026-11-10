import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { syncClient, type WsEvent } from "../api/sync";
import type { Task } from "../api/tasks";
import { invalidateTaskLists, queryKeys } from "../utils/queryKeys";
import {
  applyTaskDelete,
  applyTaskReorder,
  applyTaskUpsert,
  type TaskReorderPayload,
} from "../utils/taskCache";

/** Mount once at the app root: mirrors WebSocket task events into the React Query cache. */
export function useLiveUpdates(): void {
  const queryClient = useQueryClient();

  useEffect(() => {
    const handleUpsert = (event: WsEvent) => {
      const task = event.payload as unknown as Task;
      if (!task?.id) return;
      // Routes list items to their list bucket, never into the inbox views.
      applyTaskUpsert(queryClient, task);
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
    };

    // A drag ships one event with the new ranks, so apply them quietly.
    const handleReorder = (event: WsEvent) => {
      applyTaskReorder(queryClient, (event.payload ?? {}) as TaskReorderPayload);
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
    };

    const handleDeleted = (event: WsEvent) => {
      const { id } = (event.payload ?? {}) as { id?: string };
      if (!id) return;
      applyTaskDelete(queryClient, id);
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
    };

    const handleCompletionChanged = () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.habitCompletions });
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
    };

    // Fell too far behind the event buffer: refetch everything, list buckets included.
    const handleStale = () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasksAll });
      queryClient.invalidateQueries({ queryKey: queryKeys.taskListTasksAll });
      queryClient.invalidateQueries({ queryKey: queryKeys.habitCompletions });
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
      invalidateTaskLists(queryClient);
    };

    // A merge resolved; the payload is the winning task, so merge it like task.updated.
    const handleMerge = (event: WsEvent) => {
      handleUpsert(event);
      queryClient.invalidateQueries({ queryKey: queryKeys.scheduledTasks });
    };

    // Membership changed: refresh the list picker and everything cached for that list.
    const handleShareChanged = (event: WsEvent) => {
      invalidateTaskLists(queryClient);
      const { list_id: listId } = (event.payload ?? {}) as { list_id?: string };
      if (!listId) return;
      queryClient.invalidateQueries({ queryKey: queryKeys.taskList(listId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.taskListTasks(listId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.taskListMembers(listId) });
    };

    const offs = [
      syncClient.on("task.created", handleUpsert),
      syncClient.on("task.updated", handleUpsert),
      syncClient.on("task.reordered", handleReorder),
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
