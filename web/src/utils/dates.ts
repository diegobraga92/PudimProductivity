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
 * Get ISO date strings (YYYY-MM-DD) for the current week (Monday to Sunday),
 * using local timezone.
 */
export function getWeekDates(): string[] {
  const dates: string[] = [];
  const now = new Date();
  const dayOfWeek = now.getDay();
  const monday = new Date(now);
  monday.setDate(now.getDate() - ((dayOfWeek + 6) % 7));
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
