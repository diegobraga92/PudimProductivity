import { useEffect } from "react";
import { syncClient, type WsEvent } from "../api/sync";
import { useToast } from "../components/toastContext";
import type { Task } from "../api/tasks";

interface TaskPayload {
  id: string;
  title?: string;
  completed_date?: string;
}

function fmt(event: WsEvent): { icon: string; title: string; body?: string } | null {
  const payload = (event.payload ?? {}) as unknown as TaskPayload;
  switch (event.type) {
    case "task.created": {
      const task = payload as Task;
      if (!task.title) return null;
      return { icon: "📝", title: "New task", body: task.title };
    }
    case "task.updated": {
      const task = payload as Task;
      if (!task.title) return null;
      return { icon: "✏️", title: "Task updated", body: task.title };
    }
    case "task.deleted":
      return { icon: "🗑️", title: "Task deleted", body: "A task was removed." };
    case "task.completed":
      if (!payload.title) return null;
      return { icon: "🎉", title: "Habit completed", body: payload.title };
    case "task.uncompleted":
      if (!payload.title) return null;
      return { icon: "↩️", title: "Completion removed", body: payload.title };
    default:
      return null;
  }
}

/**
 * Shows in-app toasts for real-time task events delivered over the WebSocket
 * stream (Phase 3). Events raised by any client appear here — this is the
 * "push notification" experience on the web, replacing OS push.
 */
export function useTaskNotifier(): void {
  const { pushToast } = useToast();

  useEffect(() => {
    const notifiable: Array<Parameters<typeof syncClient.on>[0]> = [
      "task.created",
      "task.updated",
      "task.deleted",
      "task.completed",
      "task.uncompleted",
    ];

    const offs = notifiable.map((type) =>
      syncClient.on(type, (event) => {
        const t = fmt(event);
        if (t) pushToast(t);
      })
    );

    return () => offs.forEach((off) => off());
  }, [pushToast]);
}
