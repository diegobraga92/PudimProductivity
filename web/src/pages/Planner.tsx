import { useQuery } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import {
  listScheduledTasks,
  getAllTaskCompletions,
  type Task,
} from "../api/tasks";
import { getWeekDates, formatWeekRange, sanitizeTime } from "../utils/dates";

const DAYS = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"] as const;
const DAY_LABELS: Record<string, string> = {
  mon: "Mon", tue: "Tue", wed: "Wed", thu: "Thu",
  fri: "Fri", sat: "Sat", sun: "Sun",
};

// Grid hours: 6AM to 10PM (16 hours)
const HOURS = Array.from({ length: 16 }, (_, i) => {
  const h = i + 6;
  return {
    label: h > 12 ? `${h - 12}PM` : h === 12 ? "12PM" : `${h}AM`,
    value: `${String(h).padStart(2, "0")}:00`,
  };
});

function parseTimeToMinutes(t: string): number {
  const [h, m] = sanitizeTime(t).split(":").map(Number);
  return h * 60 + m;
}

function hexToRgba(hex: string, alpha: number): string {
  const r = parseInt(hex.slice(1, 3), 16);
  const g = parseInt(hex.slice(3, 5), 16);
  const b = parseInt(hex.slice(5, 7), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

interface PlannerProps {
  onNavigate: (view: string) => void;
}

export default function Planner({ onNavigate }: PlannerProps) {
  const [weekOffset, setWeekOffset] = useState(0);

  const weekDates = getWeekDates(weekOffset);
  const from = weekDates[0];
  const to = weekDates[6];

  const { data: scheduledTasks = [] } = useQuery<Task[]>({
    queryKey: ["scheduledTasks"],
    queryFn: listScheduledTasks,
  });

  // Fetch habit completions for the visible week
  const { data: allCompletions } = useQuery({
    queryKey: ["habitCompletions", from, to],
    queryFn: async () => {
      const completions = await getAllTaskCompletions(from, to);
      const results: Record<string, string[]> = {};
      for (const c of completions) {
        if (!results[c.task_id]) results[c.task_id] = [];
        results[c.task_id].push(c.completed_date);
      }
      return results;
    },
  });

  // Handle clicking an empty cell — navigate to task create with pre-filled day/time
  const handleCellClick = useCallback((day: string, time: string) => {
    const startMinutes = parseTimeToMinutes(time);
    const endMinutes = startMinutes + 60;
    const endHour = Math.floor(endMinutes / 60);
    const endMin = endMinutes % 60;
    const endTime = `${String(endHour).padStart(2, "0")}:${String(endMin).padStart(2, "0")}`;

    // Store prefill data and navigate to task create
    sessionStorage.setItem("planner_prefill", JSON.stringify({
      day,
      start_time: time,
      end_time: endTime,
    }));
    onNavigate("tasks");
  }, [onNavigate]);

  // Handle clicking an existing entry — navigate to task detail
  const handleEntryClick = useCallback((task: Task) => {
    // We can't navigate directly to a task detail from the Planner without the app state.
    // Delegate this to the app level. For now, store the task ID and navigate to tasks.
    sessionStorage.setItem("planner_task_detail", task.id);
    onNavigate("tasks");
  }, [onNavigate]);

  // Build the grid: for each day, get tasks that include that day
  function getTasksForDay(day: string): Task[] {
    return scheduledTasks.filter((t) => {
      // Habits appear on recurrence_days days
      if (t.recurrence_days && t.recurrence_days.length > 0) {
        return (t.recurrence_days as readonly string[]).includes(day);
      }
      // One-off tasks appear on their scheduled_date's weekday
      if (t.scheduled_date) {
        const date = new Date(t.scheduled_date + "T00:00:00");
        const dayOfWeek = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"][date.getDay()];
        return dayOfWeek === day;
      }
      return false;
    });
  }

  const dayLabels = weekDates.map((_, i) => DAYS[i]);

  // Check if we can navigate forward (future weeks)
  const canGoForward = weekOffset < 0;

  return (
    <div className="animate-fade-in">
      {/* Header */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: "var(--space-md)",
        }}
      >
        <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700 }}>
          📅 Weekly Planner
        </h2>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "var(--space-sm)",
          }}
        >
          <button
            className="btn btn-ghost btn-sm"
            onClick={() => setWeekOffset(weekOffset - 1)}
            aria-label="Previous week"
          >
            &larr; Prev
          </button>
          <span
            style={{
              fontSize: "var(--font-size-sm)",
              fontWeight: 600,
              color: "var(--color-text-secondary)",
              minWidth: "100px",
              textAlign: "center",
            }}
          >
            {weekOffset === 0
              ? "This Week"
              : formatWeekRange(weekDates)}
          </span>
          <button
            className="btn btn-ghost btn-sm"
            onClick={() => setWeekOffset(weekOffset + 1)}
            disabled={!canGoForward}
            aria-label="Next week"
          >
            Next &rarr;
          </button>
        </div>
      </div>

      {/* Grid Container */}
      <div
        style={{
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-md)",
          overflow: "hidden",
          background: "var(--color-surface)",
        }}
      >
        {/* Header Row */}
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "60px repeat(7, 1fr)",
          }}
        >
          <div
            style={{
              background: "var(--color-bg)",
              borderBottom: "1px solid var(--color-border)",
              borderRight: "1px solid var(--color-border)",
              padding: "0.5rem",
              textAlign: "center",
              fontWeight: 600,
              fontSize: "var(--font-size-xs)",
              color: "var(--color-text-muted)",
            }}
          >
            Time
          </div>
          {dayLabels.map((day, idx) => (
            <div
              key={day}
              style={{
                background: "var(--color-bg)",
                borderBottom: "1px solid var(--color-border)",
                borderRight: idx < 6 ? "1px solid var(--color-border)" : "none",
                padding: "0.5rem 0",
                textAlign: "center",
                fontWeight: 600,
                fontSize: "var(--font-size-xs)",
                color: "var(--color-text)",
              }}
            >
              {DAY_LABELS[day]}
              <div style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-muted)", fontWeight: 400 }}>
                {weekDates[idx]?.slice(5) ?? ""}
              </div>
            </div>
          ))}
        </div>

        {/* Scrollable body */}
        <div
          style={{
            display: "flex",
            maxHeight: "calc(100vh - 240px)",
            overflowY: "auto",
            overflowX: "hidden",
          }}
        >
          {/* Time labels column */}
          <div style={{ flex: "0 0 60px", minWidth: 0 }}>
            {HOURS.map((hour) => (
              <div
                key={hour.value}
                style={{
                  height: "60px",
                  borderRight: "1px solid var(--color-border)",
                  borderBottom: "1px solid var(--color-border-light)",
                  padding: "0 0.4rem",
                  fontSize: "var(--font-size-xs)",
                  color: "var(--color-text-muted)",
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "flex-end",
                  position: "sticky",
                  left: 0,
                  background: "var(--color-surface)",
                  zIndex: 5,
                }}
              >
                {hour.label}
              </div>
            ))}
          </div>

          {/* Day columns */}
          {dayLabels.map((day, dayIdx) => {
            const dayTasks = getTasksForDay(day);
            return (
              <div
                key={day}
                style={{
                  flex: 1,
                  minWidth: 0,
                  position: "relative",
                  borderRight: dayIdx < 6 ? "1px solid var(--color-border-light)" : "none",
                }}
              >
                {/* Clickable hour cells */}
                {HOURS.map((hour) => (
                  <div
                    key={hour.value}
                    onClick={() => handleCellClick(day, hour.value)}
                    style={{
                      height: "60px",
                      borderBottom: "1px solid var(--color-border-light)",
                      cursor: "pointer",
                      transition: "background 0.15s ease",
                    }}
                    onMouseEnter={(e) => {
                      (e.currentTarget as HTMLElement).style.background = "var(--color-border-light)";
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLElement).style.background = "";
                    }}
                  />
                ))}

                {/* Task blocks (absolute-positioned over the cells) */}
                {dayTasks.map((task) => {
                  const taskCompletions = allCompletions?.[task.id] ?? [];
                  const color = task.color || "#3B82F6";
                  const startMinutes = task.start_time ? parseTimeToMinutes(task.start_time) - 360 : 0;
                  const endMinutes = task.end_time ? parseTimeToMinutes(task.end_time) - 360 : 60;
                  const height = Math.max(endMinutes - startMinutes, 30);

                  // Check if this task is completed for a specific date (habits only)
                  const isWeekdayCompleted = (() => {
                    if (!task.recurrence_days || taskCompletions.length === 0) return false;
                    const dayIndex = DAYS.indexOf(day as typeof DAYS[number]);
                    const dateStr = weekDates[dayIndex];
                    return dateStr ? taskCompletions.includes(dateStr) : false;
                  })();

                  return (
                    <div
                      key={`${task.id}-${day}`}
                      style={{
                        position: "absolute",
                        top: `${startMinutes}px`,
                        left: "2px",
                        right: "2px",
                        height: `${height}px`,
                        background: hexToRgba(color, 0.85),
                        borderRadius: "6px",
                        padding: "2px 4px",
                        fontSize: "var(--font-size-xs)",
                        color: "#fff",
                        cursor: "pointer",
                        overflow: "hidden",
                        whiteSpace: "nowrap",
                        textOverflow: "ellipsis",
                        display: "flex",
                        alignItems: "center",
                        gap: "2px",
                        boxShadow: "0 1px 3px rgba(0,0,0,0.15)",
                        opacity: isWeekdayCompleted ? 0.7 : 1,
                        textDecoration: isWeekdayCompleted ? "line-through" : "none",
                        zIndex: 10,
                      }}
                      onClick={(e) => {
                        e.stopPropagation();
                        handleEntryClick(task);
                      }}
                      title={`${task.title} (${sanitizeTime(task.start_time)}–${sanitizeTime(task.end_time)})${isWeekdayCompleted ? " ✓ Done" : ""}`}
                      onMouseEnter={(e) => {
                        (e.currentTarget as HTMLElement).style.opacity = "0.9";
                      }}
                      onMouseLeave={(e) => {
                        (e.currentTarget as HTMLElement).style.opacity = isWeekdayCompleted ? "0.7" : "1";
                      }}
                    >
                      {isWeekdayCompleted && <span>✓</span>}
                      {task.title}
                    </div>
                  );
                })}
              </div>
            );
          })}
        </div>
      </div>

      {/* Empty state */}
      {scheduledTasks.length === 0 && (
        <div className="empty-state" style={{ marginTop: "var(--space-lg)" }}>
          <div className="empty-state-icon">📅</div>
          <p className="empty-state-text">
            No scheduled tasks yet. Click on a time slot to add your first planned activity!
          </p>
        </div>
      )}
    </div>
  );
}