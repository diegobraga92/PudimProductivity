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
} from "../api/tasks";
import WeekHeatmap from "../components/WeekHeatmap";
import StreakBadge from "../components/StreakBadge";
import ScheduleFields from "../components/ScheduleFields";
import { computeStreaks } from "../utils/streaks";
import { getToday, sanitizeTime } from "../utils/dates";
import { COLOR_PALETTE, DAY_LABELS_FULL, STREAK_HISTORY_START } from "../utils/constants";
import {
  playHabitCompletionSound,
  playTodoCompletionSound,
  playHabitUncompletionSound,
  playTodoUncompletionSound,
} from "../utils/sounds";

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
  const [color, setColor] = useState(COLOR_PALETTE[0]);
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

  const isHabit = !!(task?.recurrence_days && task.recurrence_days.length > 0);
  const today = getToday();

  // Fetch the habit's full completion history (not just the visible 7-day
  // window) so the streak can grow across calendar weeks without a hard reset.
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
    setColor(task.color || COLOR_PALETTE[0]);
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
      <div className="card loading-card">
        <p className="text-secondary">Loading task...</p>
      </div>
    );
  }

  if (fetchError) {
    return (
      <div className="card error-card" style={{ borderWidth: "3px 0 0" }}>
        <p className="error-text">Error: {(fetchError as Error).message}</p>
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
        <h2 className="page-heading" style={{ marginBottom: "var(--space-lg)" }}>
          ✏️ Edit Task
        </h2>
        <form onSubmit={handleUpdate}>
          <div style={{ marginBottom: "var(--space-md)" }}>
            <label className="form-label">
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
      <div className="flex-between" style={{ marginBottom: "var(--space-lg)" }}>
        <button className="btn btn-ghost" onClick={onBack}>
          &larr; Back
        </button>
        <div className="flex" style={{ gap: "0.5rem" }}>
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

        <div className="flex flex-wrap" style={{ gap: "var(--space-sm)" }}>
          {isHabit ? (
            <>
              <span className="badge badge-habit">Habit</span>
              <StreakBadge current={currentStreak} longest={longestStreak} />
              <span style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-muted)" }}>
                {task.recurrence_days?.map((d) => DAY_LABELS_FULL[d]).join(", ")}
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
            className="flex flex-wrap text-xs text-muted"
            style={{ gap: "var(--space-sm)", marginTop: "var(--space-sm)" }}
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
          <label className="toggle-label">
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
          className="text-xs text-muted"
          style={{ padding: "var(--space-sm) 0", borderTop: "1px solid var(--color-border-light)" }}
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