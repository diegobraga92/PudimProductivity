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
 */
function formatShortDisplay(dateStr: string): string {
  const d = new Date(dateStr + "T00:00:00");
  const months = [
    "Jan", "Feb", "Mar", "Apr", "May", "Jun",
    "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
  ];
  return `${months[d.getMonth()]} ${d.getDate()}`;
}

/**
 * Get ISO date strings (YYYY-MM-DD) for a given week (Monday to Sunday),
 * using local timezone.
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
 * Get today's date as an ISO string (YYYY-MM-DD) using local timezone.
 */
export function getToday(): string {
  return formatLocalDate(new Date());
}

/**
 * Format a week's date range into a compact display string.
 * Same-month: "1–7 Jun"
 * Cross-month: "29 Jun – 5 Jul"
 */
export function formatWeekRange(weekDates: string[]): string {
  const start = formatShortDisplay(weekDates[0]);
  const end = formatShortDisplay(weekDates[6]);
  // Check if both dates are in the same month
  const startMonth = start.split(" ")[0];
  const endMonth = end.split(" ")[0];
  if (startMonth === endMonth) {
    return `${start.split(" ")[1]}–${end.split(" ")[1]} ${startMonth}`;
  }
  return `${start} – ${end}`;
}