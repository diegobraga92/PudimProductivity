import { useState, useCallback, useEffect, useRef } from "react";
import type { RecurrenceDay } from "../api/tasks";
import { getWeekDates, getToday, formatWeekRange } from "../utils/dates";

const DAY_ORDER: RecurrenceDay[] = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
const DAY_SHORT: Record<RecurrenceDay, string> = {
  mon: "M",
  tue: "T",
  wed: "W",
  thu: "T",
  fri: "F",
  sat: "S",
  sun: "S",
};

function getDayName(dateStr: string): RecurrenceDay {
  const d = new Date(dateStr + "T00:00:00");
  const day = d.getDay();
  return DAY_ORDER[day === 0 ? 6 : day - 1];
}

interface WeekHeatmapProps {
  recurrenceDays: RecurrenceDay[];
  completions: string[];
  onToggleDay: (date: string, completed: boolean) => void;
  disabled?: boolean;
  weekOffset?: number;
  onWeekOffsetChange?: (newOffset: number) => void;
}

export default function WeekHeatmap({
  recurrenceDays,
  completions,
  onToggleDay,
  disabled = false,
  weekOffset = 0,
  onWeekOffsetChange,
}: WeekHeatmapProps) {
  const weekDates = getWeekDates(weekOffset);
  const completedSet = new Set(completions);
  const today = getToday();
  const [animatingDate, setAnimatingDate] = useState<string | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  const handleClick = useCallback(
    (date: string, isCompleted: boolean) => {
      setAnimatingDate(date);
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => setAnimatingDate(null), 200);
      onToggleDay(date, isCompleted);
    },
    [onToggleDay]
  );

  const isCurrentWeek = weekOffset === 0;

  return (
    <div>
      {/* Week navigation */}
      {onWeekOffsetChange && (
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: "var(--space-sm)",
          }}
        >
          <button
            className="btn btn-ghost"
            style={{ padding: "0.2rem 0.5rem", fontSize: "var(--font-size-sm)" }}
            onClick={() => onWeekOffsetChange(weekOffset - 1)}
            aria-label="Previous week"
          >
            &larr; Prev
          </button>
          <span
            style={{
              fontSize: "var(--font-size-sm)",
              fontWeight: 600,
              color: "var(--color-text-secondary)",
            }}
          >
            {isCurrentWeek ? "This Week" : formatWeekRange(weekDates)}
          </span>
          <button
            className="btn btn-ghost"
            style={{ padding: "0.2rem 0.5rem", fontSize: "var(--font-size-sm)" }}
            onClick={() => onWeekOffsetChange(weekOffset + 1)}
            disabled={weekOffset >= 0}
            aria-label="Next week"
          >
            Next &rarr;
          </button>
        </div>
      )}

      <div className="week-heatmap" role="group" aria-label="Weekly habit completion tracker">
        {weekDates.map((date) => {
          const dayName = getDayName(date);
          const isScheduled = recurrenceDays.includes(dayName);
          const isCompleted = completedSet.has(date);
          const isToday = date === today;

          let className = "week-day-btn";
          if (isCompleted) className += " completed";
          else if (isScheduled) className += " scheduled";
          if (isToday) className += " today";
          if (animatingDate === date) className += " animate-complete";

          const label = `${dayName} ${date}${isCompleted ? " — completed" : isScheduled ? " — scheduled" : " — not scheduled"}`;

          return (
            <button
              key={date}
              className={className}
              onClick={() => {
                if (isScheduled || isCompleted) {
                  handleClick(date, isCompleted);
                }
              }}
              disabled={disabled || (!isScheduled && !isCompleted)}
              aria-label={label}
              aria-pressed={isCompleted}
            >
              {DAY_SHORT[dayName]}
            </button>
          );
        })}
      </div>
    </div>
  );
}