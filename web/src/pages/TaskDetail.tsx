import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback } from "react";
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
import WeekHeatmap from "../components/WeekHeatmap";
import StreakBadge from "../components/StreakBadge";
import ProgressBar from "../components/ProgressBar";
import { computeStreaks } from "../utils/streaks";
import { getWeekDates, getToday } from "../utils/dates";
import { playCompletionSound } from "../utils/sounds";

const DAY_LABELS: Record<RecurrenceDay, string> = {
  mon: "Monday",
  tue: "Tuesday",
  wed: "Wednesday",
  thu: "Thursday",
  fri: "Friday",
  sat: "Saturday",
  sun: "Sunday",
};

const DAY_ORDER: RecurrenceDay[] = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];

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
  const [weekOffset, setWeekOffset] = useState(0);
  const handleWeekOffsetChange = useCallback((newOffset: number) => {
    setWeekOffset(newOffset);
  }, []);

  const {
    data: task,
    isLoading,
    error: fetchError,
  } = useQuery({
    queryKey: ["task", taskId],
    queryFn: () => getTask(taskId),
  });

  const isHabit = task?.recurrence_days && task.recurrence_days.length > 0;
  const weekDates = getWeekDates(weekOffset);
  const from = weekDates[0];
  const to = weekDates[6];

  const { data: completions = [] } = useQuery({
    queryKey: ["taskCompletions", taskId, from, to],
    queryFn: () => getTaskCompletions(taskId, from, to),
    enabled: !!task && isHabit,
  });

  const completedDates = new Set(completions.map((c) => c.completed_date));
  const completionDateStrings = completions.map((c) => c.completed_date);
  const { current: currentStreak, longest: longestStreak } = computeStreaks(completionDateStrings);

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
        await uncompleteTask(taskId, date);
      } else {
        playCompletionSound();
        await completeTask(taskId, date);
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

  if (isLoading) {
    return (
      <div className="card" style={{ textAlign: "center", padding: "var(--space-xl)" }}>
        <p style={{ color: "var(--color-text-secondary)" }}>Loading task...</p>
      </div>
    );
  }

  if (fetchError) {
    return (
      <div className="card" style={{ borderLeft: "3px solid var(--color-danger)" }}>
        <p style={{ color: "var(--color-danger)" }}>Error: {(fetchError as Error).message}</p>
        <button className="btn btn-ghost mt-sm" onClick={onBack}>
          &larr; Back
        </button>
      </div>
    );
  }

  if (!task) return null;

  if (editing) {
    return (
      <div className="animate-fade-in" style={{ maxWidth: "400px" }}>
        <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700, marginBottom: "var(--space-lg)" }}>
          ✏️ Edit Task
        </h2>
        <form onSubmit={handleUpdate}>
          <div style={{ marginBottom: "var(--space-md)" }}>
            <label
              style={{
                display: "block",
                marginBottom: "0.3rem",
                fontWeight: 600,
                fontSize: "var(--font-size-sm)",
              }}
            >
              Title
            </label>
            <input
              type="text"
              className="input"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              autoFocus
            />
          </div>

          {error && (
            <p style={{ color: "var(--color-danger)", marginBottom: "0.5rem", fontSize: "var(--font-size-sm)" }}>
              {error}
            </p>
          )}

          <div style={{ display: "flex", gap: "0.5rem" }}>
            <button
              type="submit"
              className="btn btn-primary"
              disabled={updateMutation.isPending}
            >
              {updateMutation.isPending ? "Saving..." : "Save"}
            </button>
            <button
              type="button"
              className="btn btn-ghost"
              onClick={() => setEditing(false)}
            >
              Cancel
            </button>
          </div>
        </form>
      </div>
    );
  }

  return (
    <div className="animate-fade-in" style={{ maxWidth: "500px" }}>
      {/* Header */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: "var(--space-lg)",
        }}
      >
        <button className="btn btn-ghost" onClick={onBack}>
          &larr; Back
        </button>
        <div style={{ display: "flex", gap: "0.5rem" }}>
          <button className="btn btn-ghost" onClick={startEditing}>
            ✏️ Edit
          </button>
          <button
            className="btn btn-danger"
            onClick={() => {
              if (confirm("Delete this task?")) {
                deleteMutation.mutate();
              }
            }}
          >
            🗑 Delete
          </button>
        </div>
      </div>

      {/* Task Title */}
      <div className="card" style={{ marginBottom: "var(--space-lg)" }}>
        <h2
          style={{
            fontSize: "var(--font-size-xl)",
            fontWeight: 700,
            marginBottom: "var(--space-sm)",
            textDecoration: task.status === "done" ? "line-through" : "none",
            color: task.status === "done" ? "var(--color-text-muted)" : "var(--color-text)",
            transition: "all var(--transition-fast)",
          }}
        >
          {task.title}
        </h2>

        <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)", flexWrap: "wrap" }}>
          {isHabit ? (
            <>
              <span className="badge badge-habit">Habit</span>
              <StreakBadge current={currentStreak} longest={longestStreak} />
              <span style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-muted)" }}>
                {task.recurrence_days?.map((d) => DAY_LABELS[d]).join(", ")}
              </span>
            </>
          ) : (
            <span className={`badge ${task.status === "done" ? "badge-done" : "badge-todo"}`}>
              {task.status === "done" ? "Done" : "To Do"}
            </span>
          )}
        </div>
      </div>

      {/* One-off task toggle */}
      {!isHabit && (
        <div className="card card-todo" style={{ marginBottom: "var(--space-lg)" }}>
          <label
            style={{
              display: "flex",
              alignItems: "center",
              gap: "var(--space-sm)",
              cursor: "pointer",
              fontWeight: 500,
            }}
          >
            <input
              type="checkbox"
              checked={task.status === "done"}
              onChange={() =>
                toggleMutation.mutate(task.status === "done" ? "todo" : "done")
              }
              style={{ width: "1.2rem", height: "1.2rem", accentColor: "var(--color-primary)" }}
            />
            <span>
              {task.status === "done" ? "Mark as todo" : "Mark as done"}
            </span>
          </label>
        </div>
      )}

      {/* Habit weekly calendar */}
      {isHabit && (
        <div className="card card-habit" style={{ marginBottom: "var(--space-lg)" }}>
          <h3 style={{ fontSize: "var(--font-size-base)", fontWeight: 600, marginBottom: "var(--space-md)" }}>
            📅 This Week
          </h3>

          {/* Weekly progress */}
          {task.recurrence_days && (() => {
            const today = getToday();
            const weekScheduledDates = task.recurrence_days.map(
              (day) => weekDates[DAY_ORDER.indexOf(day)]
            ).filter((d): d is string => d !== undefined);
            const weeklyTotal = weekScheduledDates.filter((d) => d <= today).length;
            const weeklyDone = completionDateStrings.filter(
              (d) => weekScheduledDates.includes(d) && d <= today
            ).length;
            const weeklyPct = weeklyTotal > 0 ? Math.round((weeklyDone / weeklyTotal) * 100) : 0;

            return (
              <div style={{ marginBottom: "var(--space-md)" }}>
                <div
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    fontSize: "var(--font-size-xs)",
                    color: "var(--color-text-muted)",
                    marginBottom: "0.2rem",
                  }}
                >
                  <span>Weekly progress</span>
                  <span>{weeklyDone}/{weeklyTotal}</span>
                </div>
                <ProgressBar value={weeklyPct} variant="habit" />
              </div>
            );
          })()}

          <WeekHeatmap
            recurrenceDays={task.recurrence_days ?? []}
            completions={completionDateStrings}
            onToggleDay={(date) => {
              habitToggleMutation.mutate(date);
            }}
            disabled={habitToggleMutation.isPending}
            weekOffset={weekOffset}
            onWeekOffsetChange={handleWeekOffsetChange}
          />
        </div>
      )}

      {/* Metadata */}
      <div
        style={{
          fontSize: "var(--font-size-xs)",
          color: "var(--color-text-muted)",
          padding: "var(--space-sm) 0",
          borderTop: "1px solid var(--color-border-light)",
        }}
      >
        <p style={{ margin: "0.2rem 0" }}>
          Created: {new Date(task.created_at).toLocaleString()}
        </p>
        <p style={{ margin: "0.2rem 0" }}>
          Updated: {new Date(task.updated_at).toLocaleString()}
        </p>
      </div>
    </div>
  );
}
