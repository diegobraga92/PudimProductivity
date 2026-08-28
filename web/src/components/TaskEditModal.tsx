import { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useConfirm } from "./useConfirm";
import {
  getTask,
  updateTask,
  deleteTask,
  completeTask,
  uncompleteTask,
  getTaskCompletions,
  type RecurrenceDay,
  type TaskStatus,
} from "../api/tasks";
import WeekHeatmap from "./WeekHeatmap";
import ScheduleFields from "./ScheduleFields";
import RecurrenceDayPicker from "./RecurrenceDayPicker";
import Modal from "./Modal";
import { computeStreaks } from "../utils/streaks";
import { getToday, sanitizeTime } from "../utils/dates";
import { useI18n } from "../i18n";
import { TasksIcon } from "../components/icons";
import { COLOR_PALETTE, STREAK_HISTORY_START } from "../utils/constants";
import {
  playHabitCompletionSound,
  playTodoCompletionSound,
  playHabitUncompletionSound,
  playTodoUncompletionSound,
} from "../utils/sounds";

interface TaskEditModalProps {
  taskId: string;
  onClose: () => void;
  onUpdated: () => void;
  onDeleted: () => void;
}

/**
 * Modal editor for a task or habit. Opens directly in edit mode (title focused)
 * with the full editable field set, plus a collapsible habit history (streak + weekly
 * heatmap).
 */
export default function TaskEditModal({
  taskId,
  onClose,
  onUpdated,
  onDeleted,
}: TaskEditModalProps) {
  const queryClient = useQueryClient();
  const confirm = useConfirm();
  const { t } = useI18n();
  const [error, setError] = useState<string | null>(null);
  const [weekOffset, setWeekOffset] = useState(0);
  const [showHistory, setShowHistory] = useState(false);

  // Edit-form state — seeded from the task once it loads.
  const [title, setTitle] = useState("");
  const [isHabit, setIsHabit] = useState(false);
  const [selectedDays, setSelectedDays] = useState<RecurrenceDay[]>([]);
  const [showSchedule, setShowSchedule] = useState(false);
  const [startTime, setStartTime] = useState("09:00");
  const [endTime, setEndTime] = useState("10:00");
  const [color, setColor] = useState(COLOR_PALETTE[0]);
  const [scheduledDate, setScheduledDate] = useState("");
  const [alarmMinutes, setAlarmMinutes] = useState("");
  const populated = useRef(false);

  const {
    data: task,
    isLoading,
    error: fetchError,
  } = useQuery({
    queryKey: ["task", taskId],
    queryFn: () => getTask(taskId),
  });

  const today = getToday();

  // Fetch the habit's full completion history (not just the visible 7-day
  // window) so the streak can grow across calendar weeks without a hard reset.
  const { data: completions = [] } = useQuery({
    queryKey: ["taskCompletions", taskId, STREAK_HISTORY_START, today],
    queryFn: () => getTaskCompletions(taskId, STREAK_HISTORY_START, today),
    enabled: !!task && !!task.recurrence_days && task.recurrence_days.length > 0,
  });

  const completedDates = new Set(completions.map((c) => c.completed_date));
  const completionDateStrings = completions.map((c) => c.completed_date);
  const { current: currentStreak, longest: longestStreak } = computeStreaks(
    completionDateStrings,
    task?.recurrence_days
  );

  // Seed the form from the fetched task once.
  useEffect(() => {
    if (!task || populated.current) return;
    populated.current = true;
    setTitle(task.title);
    setIsHabit(!!task.recurrence_days && task.recurrence_days.length > 0);
    setSelectedDays(task.recurrence_days ?? []);
    setShowSchedule(!!task.start_time);
    setStartTime(sanitizeTime(task.start_time) || "09:00");
    setEndTime(sanitizeTime(task.end_time) || "10:00");
    setColor(task.color || COLOR_PALETTE[0]);
    setScheduledDate(task.scheduled_date || "");
    setAlarmMinutes(task.alarm_minutes != null ? String(task.alarm_minutes) : "");
  }, [task]);

  const toggleDay = (day: RecurrenceDay) => {
    setSelectedDays((prev) =>
      prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day]
    );
  };

  const updateMutation = useMutation({
    mutationFn: (req: Parameters<typeof updateTask>[1]) => updateTask(taskId, req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["task", taskId] });
      onUpdated();
      onClose();
    },
    onError: (err) => setError((err as Error).message),
  });

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

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    if (!title.trim()) {
      setError(t("tasks.validation.titleRequired"));
      return;
    }

    if (isHabit && selectedDays.length === 0) {
      setError(t("tasks.validation.dayRequired"));
      return;
    }

    if (showSchedule && startTime >= endTime) {
      setError(t("tasks.validation.timeOrder"));
      return;
    }

    updateMutation.mutate({
      title: title.trim(),
      // [] converts a habit back to a one-off. A day list keeps/creates a habit.
      recurrence_days: isHabit ? selectedDays : [],
      start_time: showSchedule ? startTime : null,
      end_time: showSchedule ? endTime : null,
      color: showSchedule ? color : null,
      scheduled_date: showSchedule && !isHabit ? scheduledDate : null,
      alarm_minutes:
        showSchedule && isHabit && alarmMinutes !== "" ? Number(alarmMinutes) : null,
    });
  };

  const handleDelete = async () => {
    const ok = await confirm({
      title: t("tasks.deleteTaskTitle"),
      confirmLabel: t("tasks.delete"),
      confirmVariant: "danger",
    });
    if (ok) deleteMutation.mutate();
  };

  return (
    <Modal onClose={onClose} maxWidth={520}>
      {isLoading && <p className="text-secondary">{t("common.loadingDot")}</p>}

      {fetchError && (
        <div>
          <p className="error-text">{t("common.error")}: {(fetchError as Error).message}</p>
          <button className="btn btn-ghost mt-sm" onClick={onClose}>
            {t("common.close")}
          </button>
        </div>
      )}

      {task && (
        <>
          {/* Header */}
          <div className="flex-between" style={{ marginBottom: "var(--space-md)" }}>
            <h2 className="page-heading" style={{ margin: 0 }}>
              <TasksIcon size={24} />
              {isHabit ? t("tasks.editHabit") : t("tasks.editTask")}
            </h2>
            <button className="btn btn-ghost btn-sm" onClick={onClose} aria-label={t("a11y.close")}>
              ✕
            </button>
          </div>

          <form onSubmit={handleSubmit}>
            {/* Title */}
            <div style={{ marginBottom: "var(--space-md)" }}>
              <label className="form-label">{t("tasks.editTitle")}</label>
              <input
                type="text"
                className="input"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
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
                {t("tasks.makeHabit")}
              </label>
            </div>

            {/* Recurrence day picker */}
            {isHabit && (
              <div style={{ marginBottom: "var(--space-md)" }}>
                <label className="form-label">{t("tasks.repeatOn")}</label>
                <RecurrenceDayPicker
                  selectedDays={selectedDays}
                  onToggle={toggleDay}
                  error={error?.includes("day") ?? false}
                />
              </div>
            )}

            {/* Planner schedule */}
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

            {/* One-off status toggle */}
            {!isHabit && (
              <div className="toggle-box">
                <label className="toggle-label">
                  <input
                    type="checkbox"
                    checked={task.status === "done"}
                    onChange={() =>
                      toggleMutation.mutate(task.status === "done" ? "todo" : "done")
                    }
                    style={{ width: "1.2rem", height: "1.2rem", accentColor: "var(--color-primary)" }}
                  />
                  {task.status === "done" ? t("tasks.markTodo") : t("tasks.markDone")}
                </label>
              </div>
            )}

            {error && (
              <div className="form-error-banner" role="alert">
                {error}
              </div>
            )}

            {/* Actions */}
            <div style={{ display: "flex", gap: "0.5rem", marginTop: "var(--space-md)" }}>
              <button type="submit" className="btn btn-primary" disabled={updateMutation.isPending}>
                {updateMutation.isPending ? t("common.saving") : "💾 " + t("common.save")}
              </button>
              <button type="button" className="btn btn-ghost" onClick={onClose}>
                {t("common.cancel")}
              </button>
              <div style={{ flex: 1 }} />
              <button
                type="button"
                className="btn btn-danger"
                onClick={handleDelete}
                disabled={deleteMutation.isPending}
              >
                🗑 {t("common.delete")}
              </button>
            </div>
          </form>

          {/* Habit history — collapsible so weekly completion stays available */}
          {isHabit && (
            <div style={{ marginTop: "var(--space-lg)" }}>
              <button
                type="button"
                className="btn btn-ghost"
                style={{ width: "100%" }}
                onClick={() => setShowHistory((v) => !v)}
              >
                {showHistory ? "▾ " + t("tasks.hideHistory") : "▸ " + t("tasks.showHistory")}
                {" · 🔥 "}
                {currentStreak}
                {longestStreak > currentStreak ? ` ${t("tasks.best", { count: longestStreak })}` : ""}
              </button>
              {showHistory && (
                <div style={{ marginTop: "var(--space-sm)" }}>
                  <WeekHeatmap
                    recurrenceDays={task.recurrence_days ?? []}
                    completions={completionDateStrings}
                    onToggleDay={(date) => habitToggleMutation.mutate(date)}
                    disabled={habitToggleMutation.isPending}
                    weekOffset={weekOffset}
                    onWeekOffsetChange={setWeekOffset}
                  />
                </div>
              )}
            </div>
          )}
        </>
      )}
    </Modal>
  );
}

