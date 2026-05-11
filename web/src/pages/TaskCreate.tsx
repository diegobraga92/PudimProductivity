import { useState } from "react";
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

    setSubmitting(true);
    try {
      await createTask({
        title: title.trim(),
        recurrence_days: isHabit ? selectedDays : undefined,
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
