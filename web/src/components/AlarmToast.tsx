import { useAlarm } from "./useAlarm";
import { sanitizeTime } from "../utils/dates";

function formatRelativeFiredTime(firedAt: string): string {
  const diffMs = Date.now() - new Date(firedAt).getTime();
  const diffMin = Math.floor(diffMs / 60_000);
  if (diffMin < 1) return "just now";
  if (diffMin === 1) return "1 min ago";
  return `${diffMin} min ago`;
}

export function AlarmToast() {
  const { activeAlarms, dismissAlarm } = useAlarm();

  if (activeAlarms.length === 0) {
    return null;
  }

  return (
    <div className="alarm-toast-container" role="region" aria-label="Alarm notifications">
      {activeAlarms.map((alarm) => {
        const { task } = alarm;
        const startTime = sanitizeTime(task.start_time) || "—";

        return (
          <div key={alarm.id} className="alarm-toast-card animate-slide-in">
            <div className="alarm-toast-icon">⏰</div>
            <div className="alarm-toast-content">
              <div className="alarm-toast-title">{task.title}</div>
              <div className="alarm-toast-meta">
                Starts at {startTime} • {formatRelativeFiredTime(alarm.firedAt)}
              </div>
            </div>
            <button
              className="alarm-toast-dismiss"
              onClick={() => dismissAlarm(alarm.id)}
              aria-label={`Dismiss alarm for ${task.title}`}
            >
              ✕
            </button>
          </div>
        );
      })}
    </div>
  );
}