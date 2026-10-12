import { formatLocalDate } from "./dates";

// JS Date.getDay(): 0 = Sunday, 1 = Monday, ... 6 = Saturday
const DAY_NAMES = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"] as const;

function getDayName(dateStr: string): string {
  return DAY_NAMES[new Date(dateStr + "T00:00:00").getDay()];
}

/** Is this date a scheduled weekday? Omitted/empty scheduledDays means every day. */
export function isScheduledOn(
  dateStr: string,
  scheduledDays?: readonly string[] | null
): boolean {
  if (!scheduledDays || scheduledDays.length === 0) return true;
  return scheduledDays.includes(getDayName(dateStr));
}

/** Scheduled days strictly between two dates; omitted scheduledDays counts every day. */
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
    if (isScheduledOn(formatLocalDate(cursor), scheduledDays)) count++;
    cursor.setDate(cursor.getDate() + 1);
  }
  return count;
}

/** Current + longest streak over YYYY-MM-DD completions. A missed *scheduled* day breaks it; other
 * weekdays (e.g. ["mon", "wed", "fri"]) are skipped, so runs span calendar weeks. */
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

  const sorted = [...completions].sort();
  const dateSet = new Set(sorted);
  const uniqueDates = [...new Set(sorted)].sort();

  // Longest run of completions, broken only by a missed scheduled day.
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

  // Current run, walked back from today; non-scheduled days are skipped.
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
    } else if (!isScheduledOn(dateStr, scheduledDays)) {
      // Non-scheduled day — skip without breaking the streak.
      checkDate.setDate(checkDate.getDate() - 1);
    } else {
      break;
    }
  }

  return { current, longest };
}