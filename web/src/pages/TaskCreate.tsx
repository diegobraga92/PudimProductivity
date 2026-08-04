import { useState, useEffect } from "react";
import { createTask, type RecurrenceDay } from "../api/tasks";
import ScheduleFields from "../components/ScheduleFields";
import { COLOR_PALETTE, DAY_OPTIONS } from "../utils/constants";

interface TaskCreateProps {
  onCreated: () => void;
  onCancel: () => void;
}

export default function TaskCreate({ onCreated, onCancel }: TaskCreateProps) {
  const [title, setTitle] = useState("");
  const [isHabit, setIsHabit] = useState(false);
  const [selectedDays, setSelectedDays] = useState<RecurrenceDay[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Scheduling fields
  const [showSchedule, setShowSchedule] = useState(false);
  const [startTime, setStartTime] = useState("09:00");
  const [endTime, setEndTime] = useState("10:00");
  const [color, setColor] = useState(COLOR_PALETTE[0]);
  const [scheduledDate, setScheduledDate] = useState("");
  const [alarmMinutes, setAlarmMinutes] = useState("");

  // Check for planner prefill data
  useEffect(() => {
    const prefillJson = sessionStorage.getItem("planner_prefill");
    if (prefillJson) {
      try {
        const prefill = JSON.parse(prefillJson);
        setShowSchedule(true);
        setStartTime(prefill.start_time || "09:00");
        setEndTime(prefill.end_time || "10:00");
        setColor(COLOR_PALETTE[0]);

        // Map the day to the scheduled date for one-off tasks
        // For habits, we set the day
        if (prefill.day) {
          setSelectedDays([prefill.day as RecurrenceDay]);
        }

        sessionStorage.removeItem("planner_prefill");
      } catch {
        // Ignore parse errors
      }
    }
  }, []);

  const toggleDay = (day: RecurrenceDay) => {
    setSelectedDays((prev) =>
      prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day]
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!title.trim()) {
      setError("Title is required");
      return;
    }

    if (isHabit && selectedDays.length === 0) {
      setError("Select at least one day for the habit");
      return;
    }

    if (showSchedule && !isHabit && !scheduledDate) {
      setError("Select a date for the scheduled task");
      return;
    }

    if (showSchedule && startTime >= endTime) {
      setError("Start time must be before end time");
      return;
    }

    setSubmitting(true);
    try {
      await createTask({
        title: title.trim(),
        recurrence_days: isHabit ? selectedDays : undefined,
        start_time: showSchedule ? startTime : undefined,
        end_time: showSchedule ? endTime : undefined,
        color: showSchedule ? color : undefined,
        scheduled_date: showSchedule && !isHabit ? scheduledDate : undefined,
        alarm_minutes: showSchedule && isHabit && alarmMinutes !== "" ? Number(alarmMinutes) : undefined,
      });
      onCreated();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="animate-fade-in" style={{ maxWidth: "450px" }}>
      <h2 className="page-heading" style={{ marginBottom: "var(--space-lg)" }}>
        ✨ New Task
      </h2>

      <form onSubmit={handleSubmit}>
        <div className="card" style={{ marginBottom: "var(--space-lg)" }}>
          <div style={{ marginBottom: "var(--space-md)" }}>
            <label className="form-label">
              What do you need to do?
            </label>
            <input
              type="text"
              className="input"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="e.g. Have hair cut"
              autoFocus
            />
          </div>

          {/* Habit toggle */}
          <div className="toggle-box">
            <label className="toggle-label">
              <input
                type="checkbox"
                checked={isHabit}
                onChange={(e) => {
                  setIsHabit(e.target.checked);
                  if (!e.target.checked) setSelectedDays([]);
                }}
                className="toggle-checkbox"
                style={{ accentColor: "var(--color-habit)" }}
              />
              Make this a habit (repeats weekly)
            </label>
          </div>

          {/* Day picker */}
          {isHabit && (
            <div style={{ marginBottom: "var(--space-md)" }}>
              <label className="form-label">
                Repeat on:
              </label>
              <div style={{ display: "flex", gap: "0.3rem", flexWrap: "wrap" }}>
                {DAY_OPTIONS.map(({ value, label }) => {
                  const isSelected = selectedDays.includes(value);
                  return (
                    <button
                      key={value}
                      type="button"
                      onClick={() => toggleDay(value)}
                      style={{
                        padding: "0.4rem 0.8rem",
                        border: isSelected
                          ? "2px solid var(--color-habit)"
                          : "1.5px solid var(--color-border)",
                        borderRadius: "var(--radius-sm)",
                        background: isSelected ? "var(--color-habit-light)" : "var(--color-surface)",
                        cursor: "pointer",
                        fontWeight: isSelected ? 600 : 400,
                        color: isSelected ? "var(--color-habit)" : "var(--color-text-secondary)",
                        fontFamily: "var(--font-family)",
                        fontSize: "var(--font-size-sm)",
                        transition: "all var(--transition-fast)",
                      }}
                    >
                      {label}
                    </button>
                  );
                })}
              </div>
            </div>
          )}

          <ScheduleFields
            showSchedule={showSchedule}
            onToggleSchedule={setShowSchedule}
            startTime={startTime}
            endTime={endTime}
            color={color}
            scheduledDate={scheduledDate}
            alarmMinutes={alarmMinutes}
            isHabit={isHabit}
            onStartTimeChange={setStartTime}
            onEndTimeChange={setEndTime}
            onColorChange={setColor}
            onScheduledDateChange={setScheduledDate}
            onAlarmMinutesChange={setAlarmMinutes}
          />

          {error && (
            <p style={{ color: "var(--color-danger)", marginBottom: "0.5rem", fontSize: "var(--font-size-sm)" }}>
              {error}
            </p>
          )}
        </div>

        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button
            type="submit"
            className="btn btn-primary"
            disabled={submitting}
          >
            {submitting ? "Adding..." : "✨ Add Task"}
          </button>
          <button
            type="button"
            className="btn btn-ghost"
            onClick={onCancel}
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}