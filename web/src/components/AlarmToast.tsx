import { useAlarm } from "./useAlarm";
import { sanitizeTime } from "../utils/dates";
import { useI18n } from "../i18n";

function formatRelativeFiredTime(firedAt: string, t: (key: string, vars?: Record<string, string | number>) => string): string {
  const diffMs = Date.now() - new Date(firedAt).getTime();
  const diffMin = Math.floor(diffMs / 60_000);
  if (diffMin < 1) return t("alarm.relativeNow");
  if (diffMin === 1) return t("alarm.relativeMinute");
  return t("alarm.relativeMinutes", { count: diffMin });
}

export function AlarmToast() {
  const { activeAlarms, dismissAlarm } = useAlarm();
  const { t } = useI18n();

  if (activeAlarms.length === 0) {
    return null;
  }

  return (
    <div className="alarm-toast-container" role="region" aria-label={t("a11y.alarmNotifications")}>
      {activeAlarms.map((alarm) => {
        const { task } = alarm;
        const startTime = sanitizeTime(task.start_time) || "—";

        return (
          <div key={alarm.id} className="alarm-toast-card animate-slide-in">
            <div className="alarm-toast-icon">⏰</div>
            <div className="alarm-toast-content">
              <div className="alarm-toast-title">{task.title}</div>
              <div className="alarm-toast-meta">
                {t("alarm.startsAt", { time: startTime, relative: formatRelativeFiredTime(alarm.firedAt, t) })}
              </div>
            </div>
            <button
              className="alarm-toast-dismiss"
              onClick={() => dismissAlarm(alarm.id)}
              aria-label={t("alarm.dismiss", { title: task.title })}
            >
              ✕
            </button>
          </div>
        );
      })}
    </div>
  );
}