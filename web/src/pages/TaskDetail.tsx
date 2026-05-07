import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  getTask,
  updateTask,
  deleteTask,
  completeTask,
  uncompleteTask,
  getTaskCompletions,
  type TaskStatus,
  type RecurrenceDay,
} from "../api/tasks";

const DAY_ORDER: RecurrenceDay[] = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
const DAY_LABELS: Record<RecurrenceDay, string> = {
  mon: "Monday",
  tue: "Tuesday",
  wed: "Wednesday",
  thu: "Thursday",
  fri: "Friday",
  sat: "Saturday",
  sun: "Sunday",
};

function getWeekDates(): string[] {
  const dates: string[] = [];
  const now = new Date();
  const dayOfWeek = now.getDay();
  const monday = new Date(now);
  monday.setDate(now.getDate() - ((dayOfWeek + 6) % 7));
  for (let i = 0; i < 7; i++) {
    const d = new Date(monday);
    d.setDate(monday.getDate() + i);
    dates.push(d.toISOString().split("T")[0]);
  }
  return dates;
}

function getDayName(dateStr: string): RecurrenceDay {
  const d = new Date(dateStr + "T00:00:00");
  const day = d.getDay();
  return DAY_ORDER[day === 0 ? 6 : day - 1];
}

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
  const [error, setError] = useState<string | null>(null);

  const {
    data: task,
    isLoading,
    error: fetchError,
  } = useQuery({
    queryKey: ["task", taskId],
    queryFn: () => getTask(taskId),
  });

  const isHabit = task?.recurrence_days && task.recurrence_days.length > 0;
  const weekDates = getWeekDates();
  const from = weekDates[0];
  const to = weekDates[6];

  const { data: completions = [] } = useQuery({
    queryKey: ["taskCompletions", taskId, from, to],
    queryFn: () => getTaskCompletions(taskId, from, to),
    enabled: !!task && isHabit,
  });

  const completedDates = new Set(completions.map((c) => c.completed_date));

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

  const toggleMutation = useMutation({
    mutationFn: (newStatus: TaskStatus) => updateTask(taskId, { status: newStatus }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["task", taskId] });
      onUpdated();
    },
    onError: (err) => setError((err as Error).message),
  });

  const habitToggleMutation = useMutation({
    mutationFn: async (date: string) => {
      if (completedDates.has(date)) {
        await uncompleteTask(taskId);
      } else {
        await completeTask(taskId);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["taskCompletions"] });
    },
    onError: (err) => setError((err as Error).message),
  });

  const startEditing = () => {
    if (!task) return;
    setTitle(task.title);
    setEditing(true);
  };

  const handleUpdate = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!title.trim()) {
      setError("Title is required");
      return;
    }

    updateMutation.mutate({ title: title.trim() });
  };

  if (isLoading) return <p>Loading task...</p>;

  if (fetchError) {
    return (
      <div style={{ padding: "1rem" }}>
        <p style={{ color: "red" }}>Error: {(fetchError as Error).message}</p>
        <button
          onClick={onBack}
          style={{ padding: "0.5rem 1rem", cursor: "pointer" }}
        >
          Back
        </button>
      </div>
    );
  }

  if (!task) return null;

  if (editing) {
    return (
      <div style={{ maxWidth: "400px", margin: "0 auto", padding: "1rem" }}>
        <h2>Edit Task</h2>
        <form onSubmit={handleUpdate}>
          <div style={{ marginBottom: "1rem" }}>
            <label
              style={{
                display: "block",
                marginBottom: "0.3rem",
                fontWeight: "bold",
              }}
            >
              Title
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
              autoFocus
            />
          </div>

          {error && (
            <p style={{ color: "red", marginBottom: "0.5rem" }}>{error}</p>
          )}

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

      <h2
        style={{
          marginBottom: "0.5rem",
          textDecoration: task.status === "done" ? "line-through" : "none",
          color: task.status === "done" ? "#888" : "inherit",
        }}
      >
        {task.title}
      </h2>

      {isHabit && (
        <span
          style={{
            display: "inline-block",
            marginBottom: "1rem",
            fontSize: "0.8rem",
            color: "#007bff",
            background: "#e6f2ff",
            padding: "0.2rem 0.5rem",
            borderRadius: "3px",
          }}
        >
          Habit &middot; {task.recurrence_days?.map((d) => DAY_LABELS[d]).join(", ")}
        </span>
      )}

      {/* One-off task toggle */}
      {!isHabit && (
        <div style={{ marginBottom: "1rem" }}>
          <label
            style={{
              display: "flex",
              alignItems: "center",
              gap: "0.5rem",
              cursor: "pointer",
            }}
          >
            <input
              type="checkbox"
              checked={task.status === "done"}
              onChange={() =>
                toggleMutation.mutate(task.status === "done" ? "todo" : "done")
              }
              style={{ width: "1.2rem", height: "1.2rem" }}
            />
            <span>
              {task.status === "done" ? "Mark as todo" : "Mark as done"}
            </span>
          </label>
        </div>
      )}

      {/* Habit weekly calendar */}
      {isHabit && (
        <div style={{ marginBottom: "1rem" }}>
          <h4 style={{ margin: "0 0 0.5rem 0", color: "#555" }}>This Week</h4>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(7, 1fr)",
              gap: "0.3rem",
            }}
          >
            {weekDates.map((date) => {
              const dayName = getDayName(date);
              const isScheduled = task.recurrence_days?.includes(dayName);
              const isCompleted = completedDates.has(date);
              const isToday =
                date === new Date().toISOString().split("T")[0];

              return (
                <button
                  key={date}
                  onClick={() => {
                    if (isScheduled || isCompleted) {
                      habitToggleMutation.mutate(date);
                    }
                  }}
                  disabled={habitToggleMutation.isPending}
                  title={`${DAY_LABELS[dayName]} ${date}`}
                  style={{
                    padding: "0.5rem",
                    border: isToday ? "2px solid #007bff" : "1px solid #ddd",
                    borderRadius: "6px",
                    background: isCompleted
                      ? "#28a745"
                      : isScheduled
                      ? "#fff3cd"
                      : "#f5f5f5",
                    color: isCompleted
                      ? "white"
                      : isScheduled
                      ? "#856404"
                      : "#ccc",
                    cursor:
                      isScheduled || isCompleted ? "pointer" : "default",
                    textAlign: "center",
                    fontSize: "0.8rem",
                  }}
                >
                  <div style={{ fontWeight: "bold" }}>
                    {DAY_LABELS[dayName].slice(0, 3)}
                  </div>
                  <div style={{ fontSize: "0.7rem", marginTop: "0.2rem" }}>
                    {date.slice(5)}
                  </div>
                  <div style={{ fontSize: "1rem", marginTop: "0.2rem" }}>
                    {isCompleted ? "✓" : isScheduled ? "○" : "—"}
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      )}

      <p style={{ fontSize: "0.85rem", color: "#888" }}>
        Created: {new Date(task.created_at).toLocaleString()}
        <br />
        Updated: {new Date(task.updated_at).toLocaleString()}
      </p>
    </div>
  );
}
