import { useEffect, useState } from "react";
import { useToast } from "./toastContext";
import { useI18n } from "../i18n";

/**
 * Renders the in-app toast stack (real-time task notifications). Reuses the
 * alarm toast styles; alarms themselves are rendered by AlarmToast.
 */
export function ToastStack() {
  const { toasts, dismissToast } = useToast();
  const { t } = useI18n();
  // Toast IDs currently playing their exit animation (fade/slide out).
  const [leaving, setLeaving] = useState<Set<string>>(new Set());

  // Auto-dismiss: schedule the exit animation shortly before the provider
  // removes the toast so the removal is animated.
  useEffect(() => {
    const timers: ReturnType<typeof setTimeout>[] = [];
    for (const t of toasts) {
      const elapsed = Date.now() - new Date(t.createdAt).getTime();
      const remaining = Math.max(0, 4_800 - elapsed);
      timers.push(
        setTimeout(() => {
          setLeaving((prev) => new Set(prev).add(t.id));
        }, remaining)
      );
    }
    return () => timers.forEach((id) => clearTimeout(id));
  }, [toasts]);

  const dismiss = (id: string) => {
    setLeaving((prev) => new Set(prev).add(id));
    setTimeout(() => {
      setLeaving((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
      dismissToast(id);
    }, 180);
  };

  if (toasts.length === 0) {
    return null;
  }

  return (
    <div className="alarm-toast-container" role="region" aria-label={t("a11y.notifications")}>
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`alarm-toast-card animate-slide-in ${leaving.has(toast.id) ? "toast-exit" : ""}`}
        >
          <div className="alarm-toast-icon">{toast.icon}</div>
          <div className="alarm-toast-content">
            <div className="alarm-toast-title">{toast.title}</div>
            {toast.body && <div className="alarm-toast-meta">{toast.body}</div>}
          </div>
          <button
            className="alarm-toast-dismiss"
            onClick={() => dismiss(toast.id)}
            aria-label={t("toast.dismiss", { title: toast.title })}
          >
            ✕
          </button>
        </div>
      ))}
    </div>
  );
}
