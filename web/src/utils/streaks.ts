/**
 * Format a Date as YYYY-MM-DD using local timezone.
 */
function formatLocalDate(d: Date): string {
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

// JS Date.getDay(): 0 = Sunday, 1 = Monday, ... 6 = Saturday
const DAY_NAMES = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"] as const;

function getDayName(dateStr: string): string {
  return DAY_NAMES[new Date(dateStr + "T00:00:00").getDay()];
}

/**
 * Whether a date falls on a scheduled day for the habit.
 * When scheduledDays is omitted, every calendar day counts as scheduled.
 */
function isScheduledDay(dateStr: string, scheduledDays?: readonly string[] | null): boolean {
  if (!scheduledDays || scheduledDays.length === 0) return true;
  return scheduledDays.includes(getDayName(dateStr));
}

/**
 * Count how many scheduled days fall strictly between fromDate and toDate.
 * When scheduledDays is omitted, counts all calendar days in between.
 */
function countScheduledGapDays(
  fromDate: string,
  toDate: string,
  scheduledDays?: readonly string[] | null
): number {
  const from = new Date(fromDate + "T00:00:00");
  const to = new Date(toDate + "T00:00:00");
  let count = 0;
  const cursor = new Date(from);
  cursor.setDate(cursor.getDate() + 1);
  while (cursor < to) {
    if (isScheduledDay(formatLocalDate(cursor), scheduledDays)) count++;
    cursor.setDate(cursor.getDate() + 1);
  }
  return count;
}

/**
 * Compute current and longest streak from a list of completion dates.
 *
 * A streak is a run of completed dates where no **scheduled** day is missed.
 * Non-scheduled days (e.g. a habit that only repeats Mon/Wed/Fri) are skipped
 * and do not break the streak, so it flows continuously across calendar weeks.
 *
 * @param completions - Completion date strings in YYYY-MM-DD format.
 * @param scheduledDays - Optional weekday names the habit repeats on
 *   (e.g. ["mon", "wed", "fri"]). When omitted, every calendar day is treated
 *   as scheduled, preserving the original behavior.
 */
export function computeStreaks(
  completions: readonly string[],
  scheduledDays?: readonly string[] | null
): {
  current: number;
  longest: number;
} {
  if (completions.length === 0) return { current: 0, longest: 0 };

  // Guard against garbage input that could cause an infinite back-scan loop.
  if (scheduledDays && scheduledDays.length > 0) {
    const valid = scheduledDays.filter((d) =>
      (DAY_NAMES as readonly string[]).includes(d)
    );
    scheduledDays = valid.length > 0 ? valid : undefined;
  }

  // Sort dates ascending and deduplicate
  const sorted = [...completions].sort();
  const dateSet = new Set(sorted);
  const uniqueDates = [...new Set(sorted)].sort();

  // Longest streak: scan consecutive completed dates, breaking only when a
  // scheduled day was missed between them.
  let longest = 0;
  let tempStreak = 0;
  for (let i = 0; i < uniqueDates.length; i++) {
    if (i === 0) {
      tempStreak = 1;
    } else {
      const gapScheduledDays = countScheduledGapDays(
        uniqueDates[i - 1],
        uniqueDates[i],
        scheduledDays
      );
      tempStreak = gapScheduledDays === 0 ? tempStreak + 1 : 1;
    }
    longest = Math.max(longest, tempStreak);
  }

  // Current streak: walk backwards from today. Non-scheduled days are skipped
  // without breaking the streak.
  let current = 0;
  const today = formatLocalDate(new Date());
  const checkDate = new Date();

  // If today is not completed, don't count it (check from yesterday instead).
  if (!dateSet.has(today)) {
    checkDate.setDate(checkDate.getDate() - 1);
  }

  while (true) {
    const dateStr = formatLocalDate(checkDate);
    if (dateSet.has(dateStr)) {
      current++;
      checkDate.setDate(checkDate.getDate() - 1);
    } else if (!isScheduledDay(dateStr, scheduledDays)) {
      // Non-scheduled day — skip without breaking the streak.
      checkDate.setDate(checkDate.getDate() - 1);
    } else {
      break;
    }
  }

  return { current, longest };
}