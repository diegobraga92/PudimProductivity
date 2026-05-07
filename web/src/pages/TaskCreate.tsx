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
    <div style={{ maxWidth: "400px", margin: "0 auto", padding: "1rem" }}>
      <h2>New Task</h2>

      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: "1rem" }}>
          <label
            style={{
              display: "block",
              marginBottom: "0.3rem",
              fontWeight: "bold",
            }}
          >
            What do you need to do?
          </label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            style={{
              width: "100%",
              padding: "0.5rem",
              border: "1px solid #ccc",
              borderRadius: "4px",
              fontSize: "1rem",
            }}
            placeholder="e.g. Have hair cut"
            autoFocus
          />
        </div>

        {/* Habit toggle */}
        <div style={{ marginBottom: "1rem" }}>
          <label
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.5rem",
              cursor: "pointer",
              fontWeight: "bold",
            }}
          >
            <input
              type="checkbox"
              checked={isHabit}
              onChange={(e) => {
                setIsHabit(e.target.checked);
                if (!e.target.checked) setSelectedDays([]);
              }}
            />
            Make this a habit (repeats weekly)
          </label>
        </div>

        {/* Day picker */}
        {isHabit && (
          <div style={{ marginBottom: "1rem" }}>
            <label
              style={{
                display: "block",
                marginBottom: "0.3rem",
                fontWeight: "bold",
              }}
            >
              Repeat on:
            </label>
            <div style={{ display: "flex", gap: "0.3rem", flexWrap: "wrap" }}>
              {DAY_LABELS.map(({ value, label }) => (
                <button
                  key={value}
                  type="button"
                  onClick={() => toggleDay(value)}
                  style={{
                    padding: "0.4rem 0.7rem",
                    border: selectedDays.includes(value)
                      ? "2px solid #007bff"
                      : "1px solid #ccc",
                    borderRadius: "4px",
                    background: selectedDays.includes(value)
                      ? "#e6f2ff"
                      : "white",
                    cursor: "pointer",
                    fontWeight: selectedDays.includes(value) ? "bold" : "normal",
                  }}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>
        )}

        {error && <p style={{ color: "red", marginBottom: "0.5rem" }}>{error}</p>}

        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button
            type="submit"
            disabled={submitting}
            style={{
              padding: "0.5rem 1rem",
              background: submitting ? "#6c757d" : "#007bff",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor: submitting ? "not-allowed" : "pointer",
            }}
          >
            {submitting ? "Adding..." : "Add Task"}
          </button>
          <button
            type="button"
            onClick={onCancel}
            style={{
              padding: "0.5rem 1rem",
              background: "#6c757d",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
            }}
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}
