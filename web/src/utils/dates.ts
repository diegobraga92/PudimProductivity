/** Date -> local YYYY-MM-DD (not toISOString(), which can be off by a day at negative UTC offsets). */
export function formatLocalDate(d: Date): string {
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

/** "YYYY-MM-DD" -> "Jun 1"; monthNames localizes the abbreviations. */
function formatShortDisplay(dateStr: string, months: string[] = [
  "Jan", "Feb", "Mar", "Apr", "May", "Jun",
  "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
]): string {
  const d = new Date(dateStr + "T00:00:00");
  return `${months[d.getMonth()]} ${d.getDate()}`;
}

/** ISO dates for a local Monday–Sunday week; weekOffset shifts by whole weeks (0 = current). */
export function getWeekDates(weekOffset = 0): string[] {
  const dates: string[] = [];
  const now = new Date();
  const dayOfWeek = now.getDay();
  const monday = new Date(now);
  monday.setDate(now.getDate() - ((dayOfWeek + 6) % 7) + weekOffset * 7);
  for (let i = 0; i < 7; i++) {
    const d = new Date(monday);
    d.setDate(monday.getDate() + i);
    dates.push(formatLocalDate(d));
  }
  return dates;
}

/** ISO dates for a rolling 7-day window ending today; offset shifts it (e.g. -1 = the previous 7 days). */
export function getRollingWindowDates(offset = 0): string[] {
  const dates: string[] = [];
  const today = new Date();
  const end = new Date(today);
  end.setDate(today.getDate() + offset * 7);
  const start = new Date(end);
  start.setDate(end.getDate() - 6);
  for (let i = 0; i < 7; i++) {
    const d = new Date(start);
    d.setDate(start.getDate() + i);
    dates.push(formatLocalDate(d));
  }
  return dates;
}

/** Today as a local YYYY-MM-DD string. */
export function getToday(): string {
  return formatLocalDate(new Date());
}

/** Compact week range: "1–7 Jun" or "29 Jun – 5 Jul"; monthNames (length 12) localizes the abbreviations. */
export function formatWeekRange(weekDates: string[], monthNames?: string[]): string {
  const start = formatShortDisplay(weekDates[0], monthNames);
  const end = formatShortDisplay(weekDates[6], monthNames);
  // Check if both dates are in the same month
  const startMonth = start.split(" ")[0];
  const endMonth = end.split(" ")[0];
  if (startMonth === endMonth) {
    return `${start.split(" ")[1]}–${end.split(" ")[1]} ${startMonth}`;
  }
  return `${start} – ${end}`;
}

/** Backend time ("09:00:00") -> HH:MM for <input type="time">; "" when invalid. */
export function sanitizeTime(t: string | null | undefined): string {
  if (!t) return "";
  const parts = t.split(":");
  if (parts.length < 2) return "";
  return `${parts[0]}:${parts[1]}`;
}

/** Minutes since midnight from "HH:MM" or "HH:MM:SS"; missing parts default to 0. */
export function parseTimeToMinutes(t: string): number {
  const [h, m] = sanitizeTime(t).split(":").map(Number);
  return (h || 0) * 60 + (m || 0);
}
