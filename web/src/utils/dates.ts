/**
 * Format a Date as YYYY-MM-DD using local timezone.
 * This avoids the UTC offset issue where .toISOString() can return
 * the wrong date for users in negative UTC offsets late in the evening.
 */
function formatLocalDate(d: Date): string {
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

/**
 * Format a string like "YYYY-MM-DD" to a short display format e.g. "Jun 1".
 * Month abbreviations can be localized by passing the translated month list
 * (e.g. from the i18n dictionary); it defaults to English.
 */
function formatShortDisplay(dateStr: string, months: string[] = [
  "Jan", "Feb", "Mar", "Apr", "May", "Jun",
  "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
]): string {
  const d = new Date(dateStr + "T00:00:00");
  return `${months[d.getMonth()]} ${d.getDate()}`;
}

/**
 * Get ISO date strings (YYYY-MM-DD) for a given calendar week
 * (Monday to Sunday), using local timezone. Used by the weekly Planner grid,
 * which is anchored to fixed Monday–Sunday columns.
 *
 * @param weekOffset - Offset relative to the current week:
 *   0 (default) = current week,
 *  -1 = previous week,
 *  +1 = next week, etc.
 */
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

/**
 * Get ISO date strings (YYYY-MM-DD) for a rolling 7-day window.
 *
 * Unlike getWeekDates (anchored to Monday), this window is anchored to today:
 * offset 0 returns the last 7 days ending today, with today as the final
 * column. This keeps habit streaks flowing continuously between calendar
 * weeks — on Monday you still see the previous week's completions instead of
 * a hard reset to a blank Monday–Sunday grid.
 *
 * @param offset - Offset relative to the current window:
 *   0 (default) = last 7 days ending today,
 *  -1 = the 7 days before that,
 *  +1 = the 7 days after that (future — usually not shown).
 */
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

/**
 * Get today's date as an ISO string (YYYY-MM-DD) using local timezone.
 */
export function getToday(): string {
  return formatLocalDate(new Date());
}

/**
 * Format a week's date range into a compact display string.
 * Same-month: "1–7 Jun"
 * Cross-month: "29 Jun – 5 Jul"
 *
 * @param monthNames Optional localized month abbreviations (length 12). Falls
 *   back to English when omitted.
 */
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

/**
 * Normalize a time string from the backend (e.g. "09:00:00") to the
 * HH:MM format expected by HTML <input type="time"> elements.
 * If the input is invalid or empty, returns an empty string.
 */
export function sanitizeTime(t: string | null | undefined): string {
  if (!t) return "";
  const parts = t.split(":");
  if (parts.length < 2) return "";
  return `${parts[0]}:${parts[1]}`;
}
