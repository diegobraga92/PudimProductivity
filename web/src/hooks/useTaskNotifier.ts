import { useEffect } from "react";
import { syncClient, type WsEvent } from "../api/sync";
import { useToast } from "../components/toastContext";
import { useI18n } from "../i18n";
import type { Task } from "../api/tasks";

interface TaskPayload {
  id: string;
  title?: string;
  completed_date?: string;
}

type Translate = (key: string, vars?: Record<string, string | number>) => string;

function fmt(event: WsEvent, t: Translate): { icon: string; title: string; body?: string } | null {
  const payload = (event.payload ?? {}) as unknown as TaskPayload;
  switch (event.type) {
    case "task.created": {
      const task = payload as Task;
      if (!task.title) return null;
      return { icon: "📝", title: t("toast.newTask"), body: task.title };
    }
    case "task.updated": {
      const task = payload as Task;
      if (!task.title) return null;
      return { icon: "✏️", title: t("toast.taskUpdated"), body: task.title };
    }
    case "task.merged": {
      // Phase 8: a CRDT merge resolved. The payload is the winning task; this
      // is the "someone else changed it" signal for collaborative edits.
      const task = payload as Task;
      if (!task.title) return null;
      return { icon: "🔄", title: t("toast.merged"), body: t("toast.mergedBody", { title: task.title }) };
    }
    case "task.deleted":
      return { icon: "🗑️", title: t("toast.taskDeleted"), body: t("toast.taskDeletedBody") };
    case "task.completed":
      if (!payload.title) return null;
      return { icon: "🎉", title: t("toast.habitCompleted"), body: payload.title };
    case "task.uncompleted":
      if (!payload.title) return null;
      return { icon: "↩️", title: t("toast.completionRemoved"), body: payload.title };
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
  const { t } = useI18n();

  useEffect(() => {
    const notifiable: Array<Parameters<typeof syncClient.on>[0]> = [
      "task.created",
      "task.updated",
      "task.merged",
      "task.deleted",
      "task.completed",
      "task.uncompleted",
    ];

    const offs = notifiable.map((type) =>
      syncClient.on(type, (event) => {
        const toast = fmt(event, t);
        if (toast) pushToast(toast);
      })
    );

    return () => offs.forEach((off) => off());
  }, [pushToast, t]);
}
