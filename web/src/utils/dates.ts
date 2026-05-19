/**
 * Get ISO date strings (YYYY-MM-DD) for the current week (Monday to Sunday).
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
    dates.push(d.toISOString().split("T")[0]);
  }
  return dates;
}

/**
 * Get today's date as an ISO string (YYYY-MM-DD).
 */
export function getToday(): string {
  return new Date().toISOString().split("T")[0];
}
