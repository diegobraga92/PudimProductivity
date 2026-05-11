import type { RecurrenceDay } from "../api/tasks";

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

interface WeekHeatmapProps {
  recurrenceDays: RecurrenceDay[];
  completions: string[];
  onToggleDay: (date: string, completed: boolean) => void;
  disabled?: boolean;
}

export default function WeekHeatmap({
  recurrenceDays,
  completions,
  onToggleDay,
  disabled = false,
}: WeekHeatmapProps) {
  const weekDates = getWeekDates();
  const completedSet = new Set(completions);
  const today = new Date().toISOString().split("T")[0];

  return (
    <div className="week-heatmap">
      {weekDates.map((date) => {
        const dayName = getDayName(date);
        const isScheduled = recurrenceDays.includes(dayName);
        const isCompleted = completedSet.has(date);
        const isToday = date === today;

        let className = "week-day-btn";
        if (isCompleted) className += " completed";
        else if (isScheduled) className += " scheduled";
        if (isToday) className += " today";

        return (
          <button
            key={date}
            className={className}
            onClick={() => {
              if (isScheduled || isCompleted) {
                onToggleDay(date, isCompleted);
              }
            }}
            disabled={disabled || (!isScheduled && !isCompleted)}
            title={`${dayName} ${date}`}
          >
            {DAY_SHORT[dayName]}
          </button>
        );
      })}
    </div>
  );
}
