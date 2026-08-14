import { ALARM_OPTIONS, COLOR_PALETTE } from "../utils/constants";
import { useI18n } from "../i18n";

interface ScheduleFieldsProps {
  showSchedule: boolean;
  onToggleSchedule: (show: boolean) => void;
  startTime: string;
  endTime: string;
  color: string;
  scheduledDate: string;
  alarmMinutes: string;
  isHabit: boolean;
  onStartTimeChange: (v: string) => void;
  onEndTimeChange: (v: string) => void;
  onColorChange: (c: string) => void;
  onScheduledDateChange: (v: string) => void;
  onAlarmMinutesChange: (v: string) => void;
}

export default function ScheduleFields({
  showSchedule,
  onToggleSchedule,
  startTime,
  endTime,
  color,
  scheduledDate,
  alarmMinutes,
  isHabit,
  onStartTimeChange,
  onEndTimeChange,
  onColorChange,
  onScheduledDateChange,
  onAlarmMinutesChange,
}: ScheduleFieldsProps) {
  const { t } = useI18n();
  return (
    <>
      {/* Schedule toggle */}
      <div className="toggle-box">
        <label className="toggle-label">
          <input
            type="checkbox"
            checked={showSchedule}
            onChange={(e) => onToggleSchedule(e.target.checked)}
            className="toggle-checkbox"
          />
          {t("tasks.schedulePlanner")}
        </label>
      </div>

      {showSchedule && (
        <div
          style={{
            marginBottom: "var(--space-md)",
            padding: "var(--space-sm) var(--space-md)",
            border: "1px solid var(--color-border-light)",
            borderRadius: "var(--radius-sm)",
          }}
        >
          {/* Time range */}
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "1fr 1fr",
              gap: "var(--space-sm)",
              marginBottom: "var(--space-md)",
            }}
          >
            <div>
              <label className="form-label-xs">
                {t("tasks.start")}
              </label>
              <input
                className="input"
                type="time"
                value={startTime}
                onChange={(e) => onStartTimeChange(e.target.value)}
              />
            </div>
            <div>
              <label className="form-label-xs">
                {t("tasks.end")}
              </label>
              <input
                className="input"
                type="time"
                value={endTime}
                onChange={(e) => onEndTimeChange(e.target.value)}
              />
            </div>
          </div>

          {/* Date picker (for one-off tasks only) */}
          {!isHabit && (
            <div style={{ marginBottom: "var(--space-md)" }}>
              <label className="form-label-xs">
                {t("tasks.date")}
              </label>
              <input
                className="input"
                type="date"
                value={scheduledDate}
                onChange={(e) => onScheduledDateChange(e.target.value)}
              />
            </div>
          )}

          {/* Alarm (for habits only) */}
          {isHabit && (
            <div style={{ marginBottom: "var(--space-md)" }}>
              <label className="form-label-xs">
                {t("tasks.alarm")}
              </label>
              <select
                className="select"
                value={alarmMinutes}
                onChange={(e) => onAlarmMinutesChange(e.target.value)}
              >
                {ALARM_OPTIONS.map((opt) => (
                  <option key={opt.value} value={opt.value}>
                    {opt.value === "" ? t("alarm.none") : t("alarm.minutesBefore", { minutes: opt.value })}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Color picker */}
          <div style={{ marginBottom: 0 }}>
            <label className="form-label-xs">
              {t("tasks.color")}
            </label>
            <div style={{ display: "flex", gap: "0.4rem", flexWrap: "wrap" }}>
              {COLOR_PALETTE.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => onColorChange(c)}
                  style={{
                    width: "28px",
                    height: "28px",
                    borderRadius: "50%",
                    background: c,
                    border: color === c ? "3px solid var(--color-text)" : "2px solid transparent",
                    cursor: "pointer",
                    transition: "all var(--transition-fast)",
                    padding: 0,
                  }}
                  aria-label={t("tasks.selectColor", { color: c })}
                />
              ))}
            </div>
          </div>
        </div>
      )}
    </>
  );
}