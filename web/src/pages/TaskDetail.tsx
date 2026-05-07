import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  getTask,
  updateTask,
  deleteTask,
  type TaskStatus,
  type TaskPriority,
} from "../api/tasks";

interface TaskDetailProps {
  taskId: string;
  onUpdated: () => void;
  onDeleted: () => void;
  onBack: () => void;
}

export default function TaskDetail({
  taskId,
  onUpdated,
  onDeleted,
  onBack,
}: TaskDetailProps) {
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<TaskStatus>("todo");
  const [priority, setPriority] = useState<TaskPriority>("medium");
  const [dueDate, setDueDate] = useState("");
  const [error, setError] = useState<string | null>(null);

  const {
    data: task,
    isLoading,
    error: fetchError,
  } = useQuery({
    queryKey: ["task", taskId],
    queryFn: () => getTask(taskId),
  });

  const updateMutation = useMutation({
    mutationFn: (req: Parameters<typeof updateTask>[1]) =>
      updateTask(taskId, req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["task", taskId] });
      onUpdated();
      setEditing(false);
    },
    onError: (err) => setError((err as Error).message),
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteTask(taskId),
    onSuccess: onDeleted,
    onError: (err) => setError((err as Error).message),
  });

  const startEditing = () => {
    if (!task) return;
    setTitle(task.title);
    setDescription(task.description || "");
    setStatus(task.status);
    setPriority(task.priority);
    setDueDate(task.due_date ? task.due_date.slice(0, 16) : "");
    setEditing(true);
  };

  const handleUpdate = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!title.trim()) {
      setError("Title is required");
      return;
    }

    updateMutation.mutate({
      title: title.trim(),
      description: description.trim() || null,
      status,
      priority,
      due_date: dueDate ? new Date(dueDate).toISOString() : null,
    });
  };

  if (isLoading) return <p>Loading task...</p>;

  if (fetchError) {
    return (
      <div style={{ padding: "1rem" }}>
        <p style={{ color: "red" }}>Error: {(fetchError as Error).message}</p>
        <button onClick={onBack} style={{ padding: "0.5rem 1rem", cursor: "pointer" }}>
          Back
        </button>
      </div>
    );
  }

  if (!task) return null;

  if (editing) {
    return (
      <div style={{ maxWidth: "500px", margin: "0 auto", padding: "1rem" }}>
        <h2>Edit Task</h2>
        <form onSubmit={handleUpdate}>
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
            />
          </div>

          <div style={{ marginBottom: "1rem" }}>
            <label style={{ display: "block", marginBottom: "0.3rem", fontWeight: "bold" }}>
              Status
            </label>
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value as TaskStatus)}
              style={{
                width: "100%",
                padding: "0.5rem",
                border: "1px solid #ccc",
                borderRadius: "4px",
              }}
            >
              <option value="todo">Todo</option>
              <option value="in_progress">In Progress</option>
              <option value="done">Done</option>
            </select>
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

          {error && <p style={{ color: "red", marginBottom: "0.5rem" }}>{error}</p>}

          <div style={{ display: "flex", gap: "0.5rem" }}>
            <button
              type="submit"
              disabled={updateMutation.isPending}
              style={{
                padding: "0.5rem 1rem",
                background: updateMutation.isPending ? "#6c757d" : "#007bff",
                color: "white",
                border: "none",
                borderRadius: "4px",
                cursor: updateMutation.isPending ? "not-allowed" : "pointer",
              }}
            >
              {updateMutation.isPending ? "Saving..." : "Save"}
            </button>
            <button
              type="button"
              onClick={() => setEditing(false)}
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

  return (
    <div style={{ maxWidth: "500px", margin: "0 auto", padding: "1rem" }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: "1rem",
        }}
      >
        <button
          onClick={onBack}
          style={{
            padding: "0.3rem 0.6rem",
            background: "transparent",
            border: "1px solid #ccc",
            borderRadius: "4px",
            cursor: "pointer",
          }}
        >
          &larr; Back
        </button>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button
            onClick={startEditing}
            style={{
              padding: "0.3rem 0.6rem",
              background: "#ffc107",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
            }}
          >
            Edit
          </button>
          <button
            onClick={() => {
              if (confirm("Delete this task?")) {
                deleteMutation.mutate();
              }
            }}
            style={{
              padding: "0.3rem 0.6rem",
              background: "#dc3545",
              color: "white",
              border: "none",
              borderRadius: "4px",
              cursor: "pointer",
            }}
          >
            Delete
          </button>
        </div>
      </div>

      <h2 style={{ marginBottom: "0.5rem" }}>{task.title}</h2>

      {task.description && (
        <p style={{ color: "#555", marginBottom: "1rem", whiteSpace: "pre-wrap" }}>
          {task.description}
        </p>
      )}

      <div style={{ display: "flex", gap: "1rem", marginBottom: "1rem" }}>
        <div>
          <strong>Status:</strong>{" "}
          <span
            style={{
              display: "inline-block",
              padding: "0.15rem 0.5rem",
              borderRadius: "3px",
              fontSize: "0.85rem",
              fontWeight: "bold",
              background:
                task.status === "done"
                  ? "#e8f5e9"
                  : task.status === "in_progress"
                  ? "#fff3e0"
                  : "#e3f2fd",
              color:
                task.status === "done"
                  ? "#2e7d32"
                  : task.status === "in_progress"
                  ? "#ef6c00"
                  : "#1565c0",
            }}
          >
            {task.status.replace("_", " ")}
          </span>
        </div>
        <div>
          <strong>Priority:</strong>{" "}
          <span
            style={{
              display: "inline-block",
              padding: "0.15rem 0.5rem",
              borderRadius: "3px",
              fontSize: "0.85rem",
              fontWeight: "bold",
              textTransform: "uppercase",
              background:
                task.priority === "high"
                  ? "#ffebee"
                  : task.priority === "medium"
                  ? "#fff3e0"
                  : "#e8f5e9",
              color:
                task.priority === "high"
                  ? "#c62828"
                  : task.priority === "medium"
                  ? "#ef6c00"
                  : "#2e7d32",
            }}
          >
            {task.priority}
          </span>
        </div>
      </div>

      {task.due_date && (
        <p>
          <strong>Due:</strong>{" "}
          {new Date(task.due_date).toLocaleDateString(undefined, {
            weekday: "long",
            year: "numeric",
            month: "long",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          })}
        </p>
      )}

      <p style={{ fontSize: "0.85rem", color: "#888" }}>
        Created: {new Date(task.created_at).toLocaleString()}
        <br />
        Updated: {new Date(task.updated_at).toLocaleString()}
      </p>
    </div>
  );
}
