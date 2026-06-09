import { useQuery } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import { listTasks, getAllTaskCompletions, type Task } from "../api/tasks";
import { listTaskLists, type TaskList } from "../api/taskLists";
import ProgressBar from "../components/ProgressBar";
import { computeStreaks } from "../utils/streaks";
import { getWeekDates, getToday, formatWeekRange } from "../utils/dates";

interface DashboardProps {
  onNavigate: (view: string, taskId?: string) => void;
}

export default function Dashboard({ onNavigate }: DashboardProps) {
  const [weekOffset, setWeekOffset] = useState(0);
  const handleWeekOffsetChange = useCallback((newOffset: number) => {
    setWeekOffset(newOffset);
  }, []);

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

  const weekDates = getWeekDates(weekOffset);
  const from = weekDates[0];
  const to = weekDates[6];
  const today = getToday();
  const DAY_ORDER = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"] as const;

  // Fetch completions for all habits this week using a single batch request
  const { data: allCompletions = {} } = useQuery({
    queryKey: ["habitCompletions", from, to],
    queryFn: async () => {
      const completions = await getAllTaskCompletions(from, to);
      const results: Record<string, string[]> = {};
      for (const task of habitTasks) {
        results[task.id] = [];
      }
      for (const c of completions) {
        if (results[c.task_id] !== undefined) {
          results[c.task_id].push(c.completed_date);
        }
      }
      return results;
    },
    enabled: habitTasks.length > 0,
  });

  // Stats
  const totalTasks = todoTasks.length + habitTasks.length;
  const doneTodos = todoTasks.filter((t) => t.status === "done").length;
  const todayHabitCompletions = Object.values(allCompletions).filter((dates) =>
    dates.includes(today)
  ).length;

  // Weekly completion rate — numerator and denominator both restricted to
  // scheduled dates that have already arrived (≤ today) to avoid >100%.
  const totalScheduledDays = habitTasks.reduce((sum, t) => {
    const scheduledDays = (t.recurrence_days ?? []).filter((d) => {
      const dayIndex = DAY_ORDER.indexOf(d as typeof DAY_ORDER[number]);
      const weekDate = weekDates[dayIndex];
      return weekDate !== undefined && weekDate <= today;
    }).length;
    return sum + scheduledDays;
  }, 0);

  const totalCompletedDays = habitTasks.reduce((sum, t) => {
    const dates = allCompletions[t.id] ?? [];
    const weekScheduledDates = (t.recurrence_days ?? []).map((d) => {
      const dayIndex = DAY_ORDER.indexOf(d as typeof DAY_ORDER[number]);
      return weekDates[dayIndex];
    }).filter((d): d is string => d !== undefined);
    return sum + dates.filter((d) => weekScheduledDates.includes(d) && d <= today).length;
  }, 0);

  const weeklyRate =
    totalScheduledDays > 0
      ? Math.round((totalCompletedDays / totalScheduledDays) * 100)
      : 0;

  // Best streak across all habits
  let bestStreak = 0;
  let bestStreakName = "";
  for (const task of habitTasks) {
    const completions = allCompletions[task.id] ?? [];
    const { current } = computeStreaks(completions);
    if (current > bestStreak) {
      bestStreak = current;
      bestStreakName = task.title;
    }
  }

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
          <div className="stat-card-value">{totalTasks}</div>
          <div className="stat-card-label">Total Tasks</div>
        </div>
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: "var(--color-done)" }}>
            {doneTodos + todayHabitCompletions}
          </div>
          <div className="stat-card-label">Done Today</div>
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
        <div className="stat-card">
          <div className="stat-card-value" style={{ color: "var(--color-primary)" }}>
            {weeklyRate}%
          </div>
          <div className="stat-card-label">Weekly Rate</div>
        </div>
      </div>

      {/* Weekly Progress */}
      <div className="card" style={{ marginBottom: "var(--space-lg)" }}>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: "var(--space-sm)",
          }}
        >
          <h3 style={{ fontSize: "var(--font-size-base)", fontWeight: 600 }}>
            📊 Weekly Habit Progress
          </h3>
          <span style={{ fontSize: "var(--font-size-sm)", color: "var(--color-text-secondary)" }}>
            {totalCompletedDays}/{totalScheduledDays} days
          </span>
        </div>
        <ProgressBar value={weeklyRate} variant="habit" />
        {/* Week navigation */}
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginTop: "var(--space-sm)",
            paddingTop: "var(--space-sm)",
            borderTop: "1px solid var(--color-border-light)",
          }}
        >
          <button
            className="btn btn-ghost"
            style={{ padding: "0.2rem 0.5rem", fontSize: "var(--font-size-sm)" }}
            onClick={() => handleWeekOffsetChange(weekOffset - 1)}
            aria-label="Previous week"
          >
            &larr; Prev Week
          </button>
          <span
            style={{
              fontSize: "var(--font-size-sm)",
              fontWeight: 600,
              color: "var(--color-text-secondary)",
            }}
          >
            {weekOffset === 0
              ? "This Week"
              : formatWeekRange(weekDates)}
          </span>
          <button
            className="btn btn-ghost"
            style={{ padding: "0.2rem 0.5rem", fontSize: "var(--font-size-sm)" }}
            onClick={() => handleWeekOffsetChange(weekOffset + 1)}
            disabled={weekOffset >= 0}
            aria-label="Next week"
          >
            Next Week &rarr;
          </button>
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
        <div className="card card-todo" style={{ cursor: "pointer" }} onClick={() => onNavigate("tasks")}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginBottom: "var(--space-sm)",
            }}
          >
            <h3 style={{ fontSize: "var(--font-size-base)", fontWeight: 600 }}>
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
        <div className="card card-habit" style={{ cursor: "pointer" }} onClick={() => onNavigate("tasks")}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginBottom: "var(--space-sm)",
            }}
          >
            <h3 style={{ fontSize: "var(--font-size-base)", fontWeight: 600 }}>
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
                    }}
                  >
                    <span>{doneToday ? "✅" : "🔄"}</span>
                    <span style={{ flex: 1 }}>{task.title}</span>
                    <span style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-muted)" }}>
                      {completions.filter((d) => d <= today).length}/{task.recurrence_days?.length ?? 0}
                    </span>
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
        <div className="card card-list mt-lg" style={{ cursor: "pointer" }} onClick={() => onNavigate("tasks")}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginBottom: "var(--space-sm)",
            }}
          >
            <h3 style={{ fontSize: "var(--font-size-base)", fontWeight: 600 }}>
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
    </div>
  );
}
