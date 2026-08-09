import { useToast } from "./toastContext";

/**
 * Renders the in-app toast stack (real-time task notifications). Reuses the
 * alarm toast styles; alarms themselves are rendered by AlarmToast.
 */
export function ToastStack() {
  const { toasts, dismissToast } = useToast();

  if (toasts.length === 0) {
    return null;
  }

  return (
    <div className="alarm-toast-container" role="region" aria-label="Notifications">
      {toasts.map((toast) => (
        <div key={toast.id} className="alarm-toast-card animate-slide-in">
          <div className="alarm-toast-icon">{toast.icon}</div>
          <div className="alarm-toast-content">
            <div className="alarm-toast-title">{toast.title}</div>
            {toast.body && <div className="alarm-toast-meta">{toast.body}</div>}
          </div>
          <button
            className="alarm-toast-dismiss"
            onClick={() => dismissToast(toast.id)}
            aria-label={`Dismiss notification: ${toast.title}`}
          >
            ✕
          </button>
        </div>
      ))}
    </div>
  );
}
