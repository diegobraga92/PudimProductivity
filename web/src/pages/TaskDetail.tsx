import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import { useConfirm } from "../components/useConfirm";
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
import { computeStreaks } from "../utils/streaks";
import { getToday, sanitizeTime } from "../utils/dates";
import {
  playHabitCompletionSound,
  playTodoCompletionSound,
  playHabitUncompletionSound,
  playTodoUncompletionSound,
} from "../utils/sounds";

const DAY_LABELS: Record<RecurrenceDay, string> = {
  mon: "Monday",
  tue: "Tuesday",
  wed: "Wednesday",
  thu: "Thursday",
  fri: "Friday",
  sat: "Saturday",
  sun: "Sunday",
};

const COLOR_PALETTE = [
  "#3B82F6", "#10B981", "#F59E0B", "#EF4444", "#8B5CF6", "#EC4899",
  "#06B6D4", "#F97316", "#6366F1", "#14B8A6", "#D946EF", "#84CC16",
];

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

  // Schedule editing state
  const [showSchedule, setShowSchedule] = useState(false);
  const [startTime, setStartTime] = useState("09:00");
  const [endTime, setEndTime] = useState("10:00");
  const [color, setColor] = useState("#3B82F6");
  const [scheduledDate, setScheduledDate] = useState("");
  const [alarmMinutes, setAlarmMinutes] = useState("");

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
  const today = getToday();

  // Fetch the habit's full completion history (not just the visible 7-day
  // window) so the streak can grow across calendar weeks without a hard reset.
  const STREAK_HISTORY_START = "2020-01-01";
  const { data: completions = [] } = useQuery({
    queryKey: ["taskCompletions", taskId, STREAK_HISTORY_START, today],
    queryFn: () => getTaskCompletions(taskId, STREAK_HISTORY_START, today),
    enabled: !!task && isHabit,
  });

  const completedDates = new Set(completions.map((c) => c.completed_date));
  const completionDateStrings = completions.map((c) => c.completed_date);
  const { current: currentStreak, longest: longestStreak } = computeStreaks(
    completionDateStrings,
    task?.recurrence_days
  );

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

  const confirm = useConfirm();

  const deleteMutation = useMutation({
    mutationFn: () => deleteTask(taskId),
    onSuccess: onDeleted,
    onError: (err) => setError((err as Error).message),
  });

  const toggleMutation = useMutation({
    mutationFn: async (newStatus: TaskStatus) => {
      if (newStatus === "done") {
        playTodoCompletionSound();
      } else {
        playTodoUncompletionSound();
      }
      return updateTask(taskId, { status: newStatus });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["task", taskId] });
      onUpdated();
    },
    onError: (err) => setError((err as Error).message),
  });

  const habitToggleMutation = useMutation({
    mutationFn: async (date: string) => {
      if (completedDates.has(date)) {
        playHabitUncompletionSound();
        await uncompleteTask(taskId, date);
      } else {
        playHabitCompletionSound();
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
    setShowSchedule(!!task.start_time);
    setStartTime(sanitizeTime(task.start_time) || "09:00");
    setEndTime(sanitizeTime(task.end_time) || "10:00");
    setColor(task.color || "#3B82F6");
    setScheduledDate(task.scheduled_date || "");
    setAlarmMinutes(task.alarm_minutes != null ? String(task.alarm_minutes) : "");
    setEditing(true);
  };

  const handleUpdate = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!title.trim()) {
      setError("Title is required");
      return;
    }

    if (showSchedule && startTime >= endTime) {
      setError("Start time must be before end time");
      return;
    }

    updateMutation.mutate({
      title: title.trim(),
      start_time: showSchedule ? startTime : null,
      end_time: showSchedule ? endTime : null,
      color: showSchedule ? color : null,
      scheduled_date: showSchedule && !isHabit ? scheduledDate : null,
      alarm_minutes: showSchedule && isHabit && alarmMinutes !== "" ? Number(alarmMinutes) : null,
    });
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

          {showSchedule && (
            <div
              style={{
                marginBottom: "var(--space-md)",
                padding: "var(--space-sm) var(--space-md)",
                border: "1px solid var(--color-border-light)",
                borderRadius: "var(--radius-sm)",
              }}
            >
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

              {isHabit && (
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
                    Alarm ⏰
                  </label>
                  <select
                    className="select"
                    value={alarmMinutes}
                    onChange={(e) => setAlarmMinutes(e.target.value)}
                  >
                    <option value="">No alarm</option>
                    <option value="5">5 min before</option>
                    <option value="10">10 min before</option>
                    <option value="15">15 min before</option>
                    <option value="30">30 min before</option>
                  </select>
                </div>
              )}

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
            onClick={async () => {
              const ok = await confirm({
                title: "Delete this task?",
                confirmLabel: "Delete",
                confirmVariant: "danger",
              });
              if (ok) {
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

        {/* Schedule info */}
        {task.start_time && task.end_time && (
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "var(--space-sm)",
              marginTop: "var(--space-sm)",
              fontSize: "var(--font-size-xs)",
              color: "var(--color-text-muted)",
              flexWrap: "wrap",
            }}
          >
            <span>📅</span>
            <span>
              {sanitizeTime(task.start_time)} – {sanitizeTime(task.end_time)}
              {task.color && (
                <span
                  style={{
                    display: "inline-block",
                    width: "10px",
                    height: "10px",
                    borderRadius: "50%",
                    background: task.color,
                    marginLeft: "0.4rem",
                    verticalAlign: "middle",
                  }}
                />
              )}
              {task.alarm_minutes != null && task.alarm_minutes > 0 && (
                <span
                  style={{
                    marginLeft: "0.5rem",
                    color: "var(--color-warning)",
                    fontWeight: 600,
                  }}
                >
                  ⏰ {task.alarm_minutes} min before
                </span>
              )}
            </span>
          </div>
        )}
        {task.scheduled_date && (
          <div
            style={{
              fontSize: "var(--font-size-xs)",
              color: "var(--color-text-muted)",
              marginTop: "0.2rem",
            }}
          >
            Scheduled for: {task.scheduled_date}
          </div>
        )}
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
            🔥 Current Streak
          </h3>

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