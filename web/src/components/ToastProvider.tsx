import { useCallback, useRef, useState } from "react";
import { ToastContext, type Toast } from "./toastContext";

/** Auto-dismiss delay for non-alarm toasts. */
const TOAST_DURATION_MS = 5_000;

/**
 * Provides lightweight in-app toasts (used for real-time task notifications
 * delivered over the WebSocket stream — Phase 3). Alarm toasts remain separate.
 */
export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const nextId = useRef(1);
  const timers = useRef<Map<string, number>>(new Map());

  const dismissToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
    const timer = timers.current.get(id);
    if (timer !== undefined) {
      window.clearTimeout(timer);
      timers.current.delete(id);
    }
  }, []);

  const pushToast = useCallback(
    (toast: Omit<Toast, "id" | "createdAt">) => {
      const id = `toast-${nextId.current++}`;
      setToasts((prev) => [
        ...prev.slice(-4),
        { ...toast, id, createdAt: new Date().toISOString() },
      ]);
      timers.current.set(
        id,
        window.setTimeout(() => dismissToast(id), TOAST_DURATION_MS)
      );
    },
    [dismissToast]
  );

  return (
    <ToastContext.Provider value={{ toasts, pushToast, dismissToast }}>
      {children}
    </ToastContext.Provider>
  );
}
