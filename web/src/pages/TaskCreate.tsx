import { useState, useEffect } from "react";
import { createTask, type RecurrenceDay } from "../api/tasks";

const DAY_LABELS: { value: RecurrenceDay; label: string }[] = [
  { value: "mon", label: "Mon" },
  { value: "tue", label: "Tue" },
  { value: "wed", label: "Wed" },
  { value: "thu", label: "Thu" },
  { value: "fri", label: "Fri" },
  { value: "sat", label: "Sat" },
  { value: "sun", label: "Sun" },
];

const COLOR_PALETTE = [
  "#3B82F6", // blue
  "#10B981", // green
  "#F59E0B", // amber
  "#EF4444", // red
  "#8B5CF6", // violet
  "#EC4899", // pink
  "#06B6D4", // cyan
  "#F97316", // orange
  "#6366F1", // indigo
  "#14B8A6", // teal
  "#D946EF", // fuchsia
  "#84CC16", // lime
];

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
      <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700, marginBottom: "var(--space-lg)" }}>
        ✨ New Task
      </h2>

      <form onSubmit={handleSubmit}>
        <div className="card" style={{ marginBottom: "var(--space-lg)" }}>
          <div style={{ marginBottom: "var(--space-md)" }}>
            <label
              style={{
                display: "block",
                marginBottom: "0.3rem",
                fontWeight: 600,
                fontSize: "var(--font-size-sm)",
              }}
            >
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
          <div
            style={{
              marginBottom: "var(--space-md)",
              padding: "var(--space-sm) var(--space-md)",
              background: "var(--color-bg)",
              borderRadius: "var(--radius-sm)",
            }}
          >
            <label
              style={{
                display: "flex",
                alignItems: "center",
                gap: "0.5rem",
                cursor: "pointer",
                fontWeight: 500,
                fontSize: "var(--font-size-sm)",
              }}
            >
              <input
                type="checkbox"
                checked={isHabit}
                onChange={(e) => {
                  setIsHabit(e.target.checked);
                  if (!e.target.checked) setSelectedDays([]);
                }}
                style={{ width: "1.1rem", height: "1.1rem", accentColor: "var(--color-habit)" }}
              />
              Make this a habit (repeats weekly)
            </label>
          </div>

          {/* Day picker */}
          {isHabit && (
            <div style={{ marginBottom: "var(--space-md)" }}>
              <label
                style={{
                  display: "block",
                  marginBottom: "0.3rem",
                  fontWeight: 600,
                  fontSize: "var(--font-size-sm)",
                }}
              >
                Repeat on:
              </label>
              <div style={{ display: "flex", gap: "0.3rem", flexWrap: "wrap" }}>
                {DAY_LABELS.map(({ value, label }) => {
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

          {/* Schedule toggle */}
          <div
            style={{
              marginBottom: "var(--space-md)",
              padding: "var(--space-sm) var(--space-md)",
              background: "var(--color-bg)",
              borderRadius: "var(--radius-sm)",
            }}
          >
            <label
              style={{
                display: "flex",
                alignItems: "center",
                gap: "0.5rem",
                cursor: "pointer",
                fontWeight: 500,
                fontSize: "var(--font-size-sm)",
              }}
            >
              <input
                type="checkbox"
                checked={showSchedule}
                onChange={(e) => setShowSchedule(e.target.checked)}
                style={{ width: "1.1rem", height: "1.1rem", accentColor: "var(--color-primary)" }}
              />
              Schedule on Planner 📅
            </label>
          </div>

          {/* Planner scheduling fields */}
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
                  <label
                    style={{
                      display: "block",
                      fontSize: "var(--font-size-xs)",
                      fontWeight: 600,
                      color: "var(--color-text-secondary)",
                      marginBottom: "var(--space-xs)",
                    }}
                  >
                    Start
                  </label>
                  <input
                    className="input"
                    type="time"
                    value={startTime}
                    onChange={(e) => setStartTime(e.target.value)}
                  />
                </div>
                <div>
                  <label
                    style={{
                      display: "block",
                      fontSize: "var(--font-size-xs)",
                      fontWeight: 600,
                      color: "var(--color-text-secondary)",
                      marginBottom: "var(--space-xs)",
                    }}
                  >
                    End
                  </label>
                  <input
                    className="input"
                    type="time"
                    value={endTime}
                    onChange={(e) => setEndTime(e.target.value)}
                  />
                </div>
              </div>

              {/* Date picker (for one-off tasks only) */}
              {!isHabit && (
                <div style={{ marginBottom: "var(--space-md)" }}>
                  <label
                    style={{
                      display: "block",
                      fontSize: "var(--font-size-xs)",
                      fontWeight: 600,
                      color: "var(--color-text-secondary)",
                      marginBottom: "var(--space-xs)",
                    }}
                  >
                    Date
                  </label>
                  <input
                    className="input"
                    type="date"
                    value={scheduledDate}
                    onChange={(e) => setScheduledDate(e.target.value)}
                  />
                </div>
              )}

              {/* Color picker */}
              <div style={{ marginBottom: 0 }}>
                <label
                  style={{
                    display: "block",
                    fontSize: "var(--font-size-xs)",
                    fontWeight: 600,
                    color: "var(--color-text-secondary)",
                    marginBottom: "var(--space-xs)",
                  }}
                >
                  Color
                </label>
                <div style={{ display: "flex", gap: "0.4rem", flexWrap: "wrap" }}>
                  {COLOR_PALETTE.map((c) => (
                    <button
                      key={c}
                      type="button"
                      onClick={() => setColor(c)}
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
                      aria-label={`Select color ${c}`}
                    />
                  ))}
                </div>
              </div>
            </div>
          )}

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