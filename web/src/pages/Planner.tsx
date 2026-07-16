import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState, useCallback } from "react";
import {
  listPlannerEntries,
  createPlannerEntry,
  updatePlannerEntry,
  deletePlannerEntry,
  type PlannerEntry,
  type CreatePlannerEntryRequest,
  type UpdatePlannerEntryRequest,
} from "../api/planner";
import { getWeekDates, formatWeekRange } from "../utils/dates";

const DAYS = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"] as const;
const DAY_LABELS: Record<string, string> = {
  mon: "Mon", tue: "Tue", wed: "Wed", thu: "Thu",
  fri: "Fri", sat: "Sat", sun: "Sun",
};

// Predefined color palette for user to choose from
const COLOR_PALETTE = [
  "#3B82F6", // blue
  "#10B981", // green
  "#F59E0B", // amber
  "#EF4444", // red
  "#8B5CF6", // violet
  "#EC4899", // pink
  "#06B6D4", // cyan
  "#F97316", // orange
  "#6366F1", // indigo
  "#14B8A6", // teal
  "#D946EF", // fuchsia
  "#84CC16", // lime
];

// Grid hours: 6AM to 10PM (16 hours)
const HOURS = Array.from({ length: 16 }, (_, i) => {
  const h = i + 6;
  return {
    label: h > 12 ? `${h - 12}PM` : h === 12 ? "12PM" : `${h}AM`,
    value: `${String(h).padStart(2, "0")}:00`,
  };
});

function sanitizeTime(t: string): string {
  // <input type="time"> can return "14:00", "14:00:00", or "14:00:00.000000"
  // Normalize to HH:MM
  const parts = t.split(":");
  return `${parts[0]}:${parts[1]}`;
}

function parseTimeToMinutes(t: string): number {
  const [h, m] = sanitizeTime(t).split(":").map(Number);
  return h * 60 + m;
}

interface ModalData {
  entry?: PlannerEntry;
  day?: string;
  time?: string;
}

export default function Planner() {
  const queryClient = useQueryClient();
  const [weekOffset, setWeekOffset] = useState(0);
  const [modal, setModal] = useState<ModalData | null>(null);
  const [formTitle, setFormTitle] = useState("");
  const [formDays, setFormDays] = useState<string[]>([]);
  const [formStartTime, setFormStartTime] = useState("09:00");
  const [formEndTime, setFormEndTime] = useState("10:00");
  const [formColor, setFormColor] = useState(COLOR_PALETTE[0]);

  const weekDates = getWeekDates(weekOffset);

  const { data: entries = [] } = useQuery<PlannerEntry[]>({
    queryKey: ["plannerEntries"],
    queryFn: listPlannerEntries,
  });

  // Filter entries that have at least one day in our day-of-week labels
  const createMutation = useMutation({
    mutationFn: (req: CreatePlannerEntryRequest) => createPlannerEntry(req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["plannerEntries"] });
      closeModal();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, req }: { id: string; req: UpdatePlannerEntryRequest }) =>
      updatePlannerEntry(id, req),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["plannerEntries"] });
      closeModal();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deletePlannerEntry,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["plannerEntries"] });
      closeModal();
    },
  });

  const closeModal = useCallback(() => {
    setModal(null);
    setFormTitle("");
    setFormDays([]);
    setFormStartTime("09:00");
    setFormEndTime("10:00");
    setFormColor(COLOR_PALETTE[0]);
  }, []);

  const handleCellClick = useCallback((day: string, time: string) => {
    // Pre-fill end time 1 hour later
    const startMinutes = parseTimeToMinutes(time);
    const endMinutes = startMinutes + 60;
    const endHour = Math.floor(endMinutes / 60);
    const endMin = endMinutes % 60;
    const endTime = `${String(endHour).padStart(2, "0")}:${String(endMin).padStart(2, "0")}`;

    setModal({ day, time });
    setFormTitle("");
    setFormDays([day]);
    setFormStartTime(time);
    setFormEndTime(endTime);
    setFormColor(COLOR_PALETTE[0]);
  }, []);

  const handleEntryClick = useCallback((entry: PlannerEntry) => {
    setModal({ entry });
    setFormTitle(entry.title);
    setFormDays([...entry.days]);
    setFormStartTime(entry.start_time);
    setFormEndTime(entry.end_time);
    setFormColor(entry.color);
  }, []);

  const handleSave = useCallback(() => {
    if (!formTitle.trim()) return;
    if (formDays.length === 0) return;

    if (modal?.entry) {
      updateMutation.mutate({
        id: modal.entry.id,
        req: {
          title: formTitle.trim(),
          days: formDays,
          start_time: sanitizeTime(formStartTime),
          end_time: sanitizeTime(formEndTime),
          color: formColor,
        },
      });
    } else {
      createMutation.mutate({
        title: formTitle.trim(),
        days: formDays,
        start_time: sanitizeTime(formStartTime),
        end_time: sanitizeTime(formEndTime),
        color: formColor,
      });
    }
  }, [formTitle, formDays, formStartTime, formEndTime, formColor, modal, createMutation, updateMutation]);

  const toggleDay = useCallback((day: string) => {
    setFormDays((prev) =>
      prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day]
    );
  }, []);

  // Build the grid: for each day, get entries that include that day
  function getEntriesForDay(day: string): PlannerEntry[] {
    return entries.filter((e) => e.days.includes(day));
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

      {/* Grid Container — outer wrapper prevents overflow clipping of absolute children */}
      <div
        style={{
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-md)",
          overflow: "hidden",
          background: "var(--color-surface)",
        }}
      >
        {/* Header Row (CSS Grid row 1) */}
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

        {/* Scrollable body: time labels + day columns */}
        <div
          style={{
            display: "flex",
            maxHeight: "calc(100vh - 240px)",
            overflowY: "auto",
            overflowX: "hidden",
          }}
        >
          {/* Time labels column (sticky left) */}
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
            const dayEntries = getEntriesForDay(day);
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
                {/* Clickable hour cells (empty background) */}
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

                {/* Planner entry blocks (absolute-positioned over the cells) */}
                {dayEntries.map((entry) => (
                  <div
                    key={entry.id}
                    style={{
                      position: "absolute",
                      top: `${((parseTimeToMinutes(entry.start_time) / 60 - 6) / 16) * 100}%`,
                      left: "2px",
                      right: "2px",
                      height: `${((parseTimeToMinutes(entry.end_time) - parseTimeToMinutes(entry.start_time)) / 60 / 16) * 100}%`,
                      background: entry.color,
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
                      zIndex: 10,
                    }}
                    onClick={(e) => {
                      e.stopPropagation();
                      handleEntryClick(entry);
                    }}
                    title={`${entry.title} (${entry.start_time}–${entry.end_time})`}
                    onMouseEnter={(e) => {
                      (e.currentTarget as HTMLElement).style.opacity = "0.9";
                    }}
                    onMouseLeave={(e) => {
                      (e.currentTarget as HTMLElement).style.opacity = "1";
                    }}
                  >
                    {entry.title}
                  </div>
                ))}
              </div>
            );
          })}
        </div>
      </div>

      {/* Empty state */}
      {entries.length === 0 && (
        <div className="empty-state" style={{ marginTop: "var(--space-lg)" }}>
          <div className="empty-state-icon">📅</div>
          <p className="empty-state-text">
            No planner entries yet. Click on a time slot to add your first activity!
          </p>
        </div>
      )}

      {/* Modal Overlay */}
      {modal !== null && (
        <div
          style={{
            position: "fixed",
            inset: 0,
            background: "rgba(0,0,0,0.4)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: 1000,
          }}
          onClick={closeModal}
        >
          <div
            className="card"
            style={{
              width: "400px",
              maxWidth: "90vw",
              padding: "var(--space-lg)",
              cursor: "default",
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <h3
              style={{
                fontSize: "var(--font-size-lg)",
                fontWeight: 600,
                marginBottom: "var(--space-md)",
              }}
            >
              {modal.entry ? "Edit Entry" : "New Entry"}
            </h3>

            {/* Title */}
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
                Title
              </label>
              <input
                className="input"
                type="text"
                placeholder="e.g. Cardio"
                value={formTitle}
                onChange={(e) => setFormTitle(e.target.value)}
                autoFocus
              />
            </div>

            {/* Days */}
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
                Days
              </label>
              <div style={{ display: "flex", gap: "0.3rem", flexWrap: "wrap" }}>
                {DAYS.map((day) => (
                  <button
                    key={day}
                    type="button"
                    onClick={() => toggleDay(day)}
                    style={{
                      padding: "0.3rem 0.6rem",
                      borderRadius: "var(--radius-full)",
                      border: formDays.includes(day)
                        ? "2px solid var(--color-primary)"
                        : "1.5px solid var(--color-border)",
                      background: formDays.includes(day)
                        ? "var(--color-primary-subtle)"
                        : "var(--color-surface)",
                      color: formDays.includes(day)
                        ? "var(--color-primary)"
                        : "var(--color-text-secondary)",
                      fontSize: "var(--font-size-xs)",
                      fontWeight: formDays.includes(day) ? 600 : 400,
                      cursor: "pointer",
                      transition: "all var(--transition-fast)",
                    }}
                  >
                    {DAY_LABELS[day]}
                  </button>
                ))}
              </div>
            </div>

            {/* Time range */}
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
                  value={formStartTime}
                  onChange={(e) => setFormStartTime(e.target.value)}
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
                  value={formEndTime}
                  onChange={(e) => setFormEndTime(e.target.value)}
                />
              </div>
            </div>

            {/* Color picker */}
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
                Color
              </label>
              <div style={{ display: "flex", gap: "0.4rem", flexWrap: "wrap" }}>
                {COLOR_PALETTE.map((c) => (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setFormColor(c)}
                    style={{
                      width: "28px",
                      height: "28px",
                      borderRadius: "50%",
                      background: c,
                      border: formColor === c ? "3px solid var(--color-text)" : "2px solid transparent",
                      cursor: "pointer",
                      transition: "all var(--transition-fast)",
                      padding: 0,
                    }}
                    aria-label={`Select color ${c}`}
                  />
                ))}
              </div>
            </div>

            {/* Actions */}
            <div
              style={{
                display: "flex",
                gap: "var(--space-sm)",
                justifyContent: "flex-end",
                borderTop: "1px solid var(--color-border-light)",
                paddingTop: "var(--space-md)",
              }}
            >
              {modal.entry && (
                <button
                  className="btn btn-danger btn-sm"
                  onClick={() => {
                    if (confirm("Delete this entry?")) {
                      deleteMutation.mutate(modal.entry!.id);
                    }
                  }}
                >
                  Delete
                </button>
              )}
              <button className="btn btn-ghost btn-sm" onClick={closeModal}>
                Cancel
              </button>
              <button
                className="btn btn-primary btn-sm"
                onClick={handleSave}
                disabled={
                  !formTitle.trim() ||
                  formDays.length === 0 ||
                  createMutation.isPending ||
                  updateMutation.isPending
                }
              >
                {createMutation.isPending || updateMutation.isPending
                  ? "Saving..."
                  : modal.entry
                    ? "Update"
                    : "Create"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}