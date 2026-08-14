import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { listTasks, type Task } from "../api/tasks";
import { listTaskLists, type TaskList } from "../api/taskLists";
import { exportBackup, importBackup } from "../api/backup";
import { useHabitCompletions } from "../hooks/useHabitCompletions";
import { computeStreaks, isScheduledOn } from "../utils/streaks";
import { getToday } from "../utils/dates";
import { useConfirm } from "../components/useConfirm";
import { useToast } from "../components/toastContext";

interface DashboardProps {
  onNavigate: (view: string, taskId?: string) => void;
}

export default function Dashboard({ onNavigate }: DashboardProps) {
  const queryClient = useQueryClient();
  const { pushToast } = useToast();
  const confirm = useConfirm();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);

  const { data: todoTasks = [] } = useQuery<Task[]>({
    queryKey: ["tasks", "one-off"],
    queryFn: () => listTasks(undefined, "one-off"),
  });

  const { data: habitTasks = [] } = useQuery<Task[]>({
    queryKey: ["tasks", "habit"],
    queryFn: () => listTasks(undefined, "habit"),
  });

  const { data: taskLists = [] } = useQuery<TaskList[]>({
    queryKey: ["taskLists"],
    queryFn: listTaskLists,
  });

  const today = getToday();
  const allCompletions = useHabitCompletions(habitTasks);

  // Stats — one-off tasks and habits are deliberately kept separate: habits are
  // daily commitments (done per-day, tracked via completions), while one-off
  // tasks are single items with a flat todo/done status and no "done today".
  const openTodos = todoTasks.filter((t) => t.status === "todo").length;
  const todayHabitCompletions = Object.values(allCompletions).filter((dates) =>
    dates.includes(today)
  ).length;
  // Habits that are due today (or were completed today even if off-schedule,
  // mirroring the mobile widgets) — the denominator for the "Habits Today" stat.
  const habitsToday = habitTasks.filter(
    (h) =>
      isScheduledOn(today, h.recurrence_days) ||
      (allCompletions[h.id] ?? []).includes(today)
  );
  const habitsScheduledToday = habitsToday.length;

  // Best streak across all habits
  let bestStreak = 0;
  let bestStreakName = "";
  for (const task of habitTasks) {
    const completions = allCompletions[task.id] ?? [];
    const { current } = computeStreaks(completions, task.recurrence_days);
    if (current > bestStreak) {
      bestStreak = current;
      bestStreakName = task.title;
    }
  }

  // Backup & Restore — export a JSON snapshot of all non-sensitive data, or
  // restore from a previous one (destructive, so it needs confirmation).
  const exportMutation = useMutation({
    mutationFn: exportBackup,
    onSuccess: ({ blob, filename }) => {
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
      pushToast({ icon: "💾", title: "Backup exported", body: `${filename} downloaded` });
    },
    onError: (err: Error) =>
      pushToast({ icon: "⚠️", title: "Export failed", body: err.message }),
  });

  const importMutation = useMutation({
    mutationFn: (file: File) => importBackup(file),
    onSuccess: (result) => {
      const total = Object.values(result.row_counts).reduce((sum, n) => sum + n, 0);
      pushToast({ icon: "♻️", title: "Backup restored", body: `${total} rows restored` });
      // A restore replaces everything the client may have cached.
      queryClient.invalidateQueries();
      setSelectedFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
    },
    onError: (err: Error) =>
      pushToast({ icon: "⚠️", title: "Restore failed", body: err.message }),
  });

  const handleFileSelected = async (file: File) => {
    setSelectedFile(file);
    const ok = await confirm({
      title: "Restore backup?",
      message: `Importing "${file.name}" will REPLACE all current tasks, lists, recipes, library, meal plans and insights. This cannot be undone. Continue?`,
      confirmLabel: "Restore",
      confirmVariant: "danger",
    });
    if (ok) {
      importMutation.mutate(file);
    } else {
      setSelectedFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  return (
    <div className="animate-fade-in">
      {/* Stats Grid */}
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
          gap: "var(--space-md)",
          marginBottom: "var(--space-lg)",
        }}
      >
        <div className="stat-card">
          <div className="stat-card-value">{openTodos}</div>
          <div className="stat-card-label">Open Tasks</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: "var(--color-habit)" }}>
            {todayHabitCompletions}/{habitsScheduledToday}
          </div>
          <div className="stat-card-label">Habits Today</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: "var(--color-habit)" }}>
            {bestStreak > 0 ? `🔥${bestStreak}` : "—"}
          </div>
          <div className="stat-card-label">
            {bestStreakName
              ? `Best Streak: ${bestStreakName.length > 12 ? bestStreakName.slice(0, 12) + "…" : bestStreakName}`
              : "Best Streak"}
          </div>
        </div>
      </div>

      {/* Quick Overview Sections */}
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: "var(--space-md)",
        }}
      >
        {/* Todos Overview */}
        <div className="card card-todo card-clickable" onClick={() => onNavigate("tasks")}>
          <div className="section-card-header">
            <h3 className="card-title">
              📋 To-Dos
            </h3>
            <span className="badge badge-todo">{todoTasks.length}</span>
          </div>
          {todoTasks.length === 0 ? (
            <p className="empty-state-text" style={{ margin: 0 }}>
              No todos yet. Create one!
            </p>
          ) : (
            <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
              {todoTasks.slice(0, 5).map((task) => (
                <li
                  key={task.id}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: "var(--space-sm)",
                    padding: "0.3rem 0",
                    borderBottom: "1px solid var(--color-border-light)",
                    fontSize: "var(--font-size-sm)",
                    textDecoration: task.status === "done" ? "line-through" : "none",
                    color: task.status === "done" ? "var(--color-text-muted)" : "var(--color-text)",
                  }}
                >
                  <span>{task.status === "done" ? "✅" : "⬜"}</span>
                  <span>{task.title}</span>
                </li>
              ))}
              {todoTasks.length > 5 && (
                <li style={{ padding: "0.3rem 0", fontSize: "var(--font-size-xs)", color: "var(--color-primary)" }}>
                  +{todoTasks.length - 5} more...
                </li>
              )}
            </ul>
          )}
        </div>

        {/* Habits Overview */}
        <div className="card card-habit card-clickable" onClick={() => onNavigate("tasks")}>
          <div className="section-card-header">
            <h3 className="card-title">
              🔄 Habits
            </h3>
            <span className="badge badge-habit">{habitTasks.length}</span>
          </div>
          {habitTasks.length === 0 ? (
            <p className="empty-state-text" style={{ margin: 0 }}>
              No habits yet. Create one!
            </p>
          ) : (
            <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
              {habitTasks.slice(0, 5).map((task) => {
                const completions = allCompletions[task.id] ?? [];
                const doneToday = completions.includes(today);
                const scheduledToday = isScheduledOn(today, task.recurrence_days);
                const offSchedule = !scheduledToday && !doneToday;
                const { current: streak } = computeStreaks(completions, task.recurrence_days);
                return (
                  <li
                    key={task.id}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: "var(--space-sm)",
                      padding: "0.3rem 0",
                      borderBottom: "1px solid var(--color-border-light)",
                      fontSize: "var(--font-size-sm)",
                      opacity: offSchedule ? 0.5 : 1,
                    }}
                  >
                    <span>{doneToday ? "✅" : scheduledToday ? "🔄" : "⏸️"}</span>
                    <span style={{ flex: 1 }}>{task.title}</span>
                    {streak > 0 && (
                      <span style={{ fontSize: "var(--font-size-xs)", fontWeight: 600, color: "var(--color-habit)" }}>
                        🔥 {streak}
                      </span>
                    )}
                  </li>
                );
              })}
              {habitTasks.length > 5 && (
                <li style={{ padding: "0.3rem 0", fontSize: "var(--font-size-xs)", color: "var(--color-primary)" }}>
                  +{habitTasks.length - 5} more...
                </li>
              )}
            </ul>
          )}
        </div>
      </div>

      {/* Lists Overview */}
      {taskLists.length > 0 && (
        <div className="card card-list mt-lg card-clickable" onClick={() => onNavigate("tasks")}>
          <div className="section-card-header">
            <h3 className="card-title">
              📁 Task Lists
            </h3>
            <span className="badge" style={{ background: "var(--color-list-light)", color: "var(--color-list)" }}>
              {taskLists.length}
            </span>
          </div>
          <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-sm)" }}>
            {taskLists.map((list) => (
              <span
                key={list.id}
                style={{
                  padding: "0.2rem 0.6rem",
                  background: "var(--color-list-light)",
                  color: "var(--color-list)",
                  borderRadius: "var(--radius-full)",
                  fontSize: "var(--font-size-xs)",
                  fontWeight: 600,
                }}
              >
                {list.name}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Backup & Restore */}
      <div className="card mt-lg">
        <div className="section-card-header">
          <h3 className="card-title">💾 Backup &amp; Restore</h3>
        </div>
        <p className="text-sm text-secondary" style={{ margin: "0 0 var(--space-md)" }}>
          Download all your non-sensitive data (tasks, lists, recipes, library, meal plans and
          insights) as a JSON file, or restore from a previous backup. Restoring replaces the
          current data — it cannot be undone.
        </p>
        <div className="flex-center" style={{ gap: "var(--space-md)", flexWrap: "wrap" }}>
          <button
            className="btn btn-primary"
            onClick={() => exportMutation.mutate()}
            disabled={exportMutation.isPending}
          >
            {exportMutation.isPending ? "Exporting…" : "⬇️ Export backup"}
          </button>
          <button
            className="btn btn-ghost"
            onClick={() => fileInputRef.current?.click()}
            disabled={importMutation.isPending}
          >
            {importMutation.isPending ? "Restoring…" : "⬆️ Restore backup"}
          </button>
          <input
            ref={fileInputRef}
            type="file"
            accept="application/json,.json"
            style={{ display: "none" }}
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) void handleFileSelected(file);
            }}
          />
        </div>
        {selectedFile && !importMutation.isPending && (
          <p className="text-sm text-secondary" style={{ margin: "var(--space-sm) 0 0" }}>
            Selected file: {selectedFile.name}
          </p>
        )}
      </div>
    </div>
  );
}
