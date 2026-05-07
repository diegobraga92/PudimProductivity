import { useState } from "react";
import { createTask, type TaskPriority } from "../api/tasks";

interface TaskCreateProps {
  onCreated: () => void;
  onCancel: () => void;
}

export default function TaskCreate({ onCreated, onCancel }: TaskCreateProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [priority, setPriority] = useState<TaskPriority>("medium");
  const [dueDate, setDueDate] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!title.trim()) {
      setError("Title is required");
      return;
    }

    setSubmitting(true);
    try {
      await createTask({
        title: title.trim(),
        description: description.trim() || null,
        priority,
        due_date: dueDate ? new Date(dueDate).toISOString() : null,
      });
      onCreated();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div style={{ maxWidth: "500px", margin: "0 auto", padding: "1rem" }}>
      <h2>Create Task</h2>

      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: "1rem" }}>
          <label style={{ display: "block", marginBottom: "0.3rem", fontWeight: "bold" }}>
            Title *
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
            }}
            placeholder="What needs to be done?"
            autoFocus
          />
        </div>

        <div style={{ marginBottom: "1rem" }}>
          <label style={{ display: "block", marginBottom: "0.3rem", fontWeight: "bold" }}>
            Description
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            style={{
              width: "100%",
              padding: "0.5rem",
              border: "1px solid #ccc",
              borderRadius: "4px",
              minHeight: "80px",
            }}
            placeholder="Optional details..."
          />
        </div>

        <div style={{ marginBottom: "1rem" }}>
          <label style={{ display: "block", marginBottom: "0.3rem", fontWeight: "bold" }}>
            Priority
          </label>
          <select
            value={priority}
            onChange={(e) => setPriority(e.target.value as TaskPriority)}
            style={{
              width: "100%",
              padding: "0.5rem",
              border: "1px solid #ccc",
              borderRadius: "4px",
            }}
          >
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
          </select>
        </div>

        <div style={{ marginBottom: "1rem" }}>
          <label style={{ display: "block", marginBottom: "0.3rem", fontWeight: "bold" }}>
            Due Date
          </label>
          <input
            type="datetime-local"
            value={dueDate}
            onChange={(e) => setDueDate(e.target.value)}
            style={{
              width: "100%",
              padding: "0.5rem",
              border: "1px solid #ccc",
              borderRadius: "4px",
            }}
          />
        </div>

        {error && (
          <p style={{ color: "red", marginBottom: "0.5rem" }}>{error}</p>
        )}

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
            {submitting ? "Creating..." : "Create Task"}
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
