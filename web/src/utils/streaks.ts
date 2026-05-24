/**
 * Compute current and longest streak from a sorted list of completion dates.
 * A streak is consecutive days (including today if completed).
 */
export function computeStreaks(completions: string[]): {
  current: number;
  longest: number;
} {
  if (completions.length === 0) return { current: 0, longest: 0 };

  // Sort dates ascending
  const sorted = [...completions].sort();

  // Build a set for O(1) lookup
  const dateSet = new Set(sorted);

  // Find the longest streak by scanning consecutive days
  let longest = 0;
  let tempStreak = 0;
  const uniqueDates = [...new Set(sorted)].sort();

  for (let i = 0; i < uniqueDates.length; i++) {
    if (i === 0) {
      tempStreak = 1;
    } else {
      const prev = new Date(uniqueDates[i - 1]);
      const curr = new Date(uniqueDates[i]);
      const diffMs = curr.getTime() - prev.getTime();
      const diffDays = Math.round(diffMs / (1000 * 60 * 60 * 24));
      if (diffDays === 1) {
        tempStreak++;
      } else {
        tempStreak = 1;
      }
    }
    longest = Math.max(longest, tempStreak);
  }

  // Compute current streak: count backwards from today
  let current = 0;
  const today = new Date().toISOString().split("T")[0];

  // If today is not completed, check if yesterday was (for "active" streak)
  const checkDate = new Date();
  if (!dateSet.has(today)) {
    // Don't count today if not completed, but check if yesterday was
    checkDate.setDate(checkDate.getDate() - 1);
  }

  while (true) {
    const dateStr = checkDate.toISOString().split("T")[0];
    if (dateSet.has(dateStr)) {
      current++;
      checkDate.setDate(checkDate.getDate() - 1);
    } else {
      break;
    }
  }

  return { current, longest };
}
