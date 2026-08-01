import { describe, it, expect, vi, afterEach } from "vitest";
import { computeStreaks } from "./streaks";

// Helper: freeze Date.now() and new Date() to a fixed instant.
// Uses noon local time to avoid timezone boundary issues with formatLocalDate().
function mockToday(isoDate: string) {
  const [y, m, d] = isoDate.split("-").map(Number);
  const fixed = new Date(y, m - 1, d, 12, 0, 0, 0); // noon local time
  vi.useFakeTimers();
  vi.setSystemTime(fixed);
}

afterEach(() => {
  vi.useRealTimers();
});

describe("computeStreaks — empty input", () => {
  it("returns {current:0, longest:0} for empty array", () => {
    expect(computeStreaks([])).toEqual({ current: 0, longest: 0 });
  });
});

describe("computeStreaks — single completion", () => {
  it("returns {current:1, longest:1} when the only completion is today", () => {
    mockToday("2026-05-25");
    expect(computeStreaks(["2026-05-25"])).toEqual({ current: 1, longest: 1 });
  });

  it("returns {current:1, longest:1} when the only completion is yesterday", () => {
    mockToday("2026-05-25");
    expect(computeStreaks(["2026-05-24"])).toEqual({ current: 1, longest: 1 });
  });

  it("returns {current:0, longest:1} when the only completion was two days ago", () => {
    mockToday("2026-05-25");
    expect(computeStreaks(["2026-05-23"])).toEqual({ current: 0, longest: 1 });
  });
});

describe("computeStreaks — consecutive streaks", () => {
  it("counts consecutive days including today", () => {
    mockToday("2026-05-25");
    const completions = ["2026-05-23", "2026-05-24", "2026-05-25"];
    expect(computeStreaks(completions)).toEqual({ current: 3, longest: 3 });
  });

  it("counts consecutive days ending yesterday", () => {
    mockToday("2026-05-25");
    const completions = ["2026-05-22", "2026-05-23", "2026-05-24"];
    expect(computeStreaks(completions)).toEqual({ current: 3, longest: 3 });
  });

  it("does not count streak that ended two days ago", () => {
    mockToday("2026-05-25");
    const completions = ["2026-05-21", "2026-05-22", "2026-05-23"];
    expect(computeStreaks(completions)).toEqual({ current: 0, longest: 3 });
  });

  it("handles input in unsorted order", () => {
    mockToday("2026-05-25");
    const completions = ["2026-05-25", "2026-05-23", "2026-05-24"];
    expect(computeStreaks(completions)).toEqual({ current: 3, longest: 3 });
  });
});

describe("computeStreaks — broken streak", () => {
  it("tracks longest across two separate runs", () => {
    mockToday("2026-05-25");
    // Run A: 3 days (May 1–3), Run B: 2 days (May 24–25)
    const completions = [
      "2026-05-01",
      "2026-05-02",
      "2026-05-03",
      "2026-05-24",
      "2026-05-25",
    ];
    const result = computeStreaks(completions);
    expect(result.longest).toBe(3);
    expect(result.current).toBe(2);
  });

  it("ignores a gap day in the current run", () => {
    mockToday("2026-05-25");
    // May 23 is missing
    const completions = ["2026-05-22", "2026-05-24", "2026-05-25"];
    expect(computeStreaks(completions)).toEqual({ current: 2, longest: 2 });
  });
});

describe("computeStreaks — duplicate completions", () => {
  it("deduplicates dates before computing streaks", () => {
    mockToday("2026-05-25");
    const completions = ["2026-05-25", "2026-05-25", "2026-05-24"];
    // Should behave like ["2026-05-24", "2026-05-25"]
    expect(computeStreaks(completions)).toEqual({ current: 2, longest: 2 });
  });
});

describe("computeStreaks — timezone boundary edge case", () => {
  it("uses UTC date string consistently (no local-TZ offset drift)", () => {
    // Simulate a user at UTC-12 where local midnight lags UTC by 12 hours.
    // The backend always stores dates as UTC ISO strings (YYYY-MM-DD).
    // As long as the client compares UTC ISO strings to UTC ISO strings
    // the streak is correct regardless of local TZ.
    mockToday("2026-05-25"); // Noon UTC — local UTC-12 would still be May 24
    // Backend sends these UTC date strings:
    const completions = ["2026-05-24", "2026-05-25"];
    const result = computeStreaks(completions);
    // "today" inside computeStreaks resolves to 2026-05-25 UTC → streak includes it
    expect(result.current).toBe(2);
  });
});

describe("computeStreaks — scheduled days (habits)", () => {
  // Jan 2026: 5th=Mon, 7th=Wed, 9th=Fri, 12th=Mon, 14th=Wed, 16th=Fri,
  //           19th=Mon, 21st=Wed, 23rd=Fri, 24th=Sat

  it("keeps streak flowing across calendar weeks for a Mon/Wed/Fri habit", () => {
    mockToday("2026-01-23"); // Friday — 6 scheduled days done over 2 weeks
    const completions = [
      "2026-01-12", // Mon
      "2026-01-14", // Wed
      "2026-01-16", // Fri
      "2026-01-19", // Mon
      "2026-01-21", // Wed
      "2026-01-23", // Fri
    ];
    expect(computeStreaks(completions, ["mon", "wed", "fri"]))
      .toEqual({ current: 6, longest: 6 });
  });

  it("skips non-scheduled days when today is not scheduled", () => {
    mockToday("2026-01-24"); // Saturday — not a scheduled day
    const completions = [
      "2026-01-12",
      "2026-01-14",
      "2026-01-16",
      "2026-01-19",
      "2026-01-21",
      "2026-01-23",
    ];
    expect(computeStreaks(completions, ["mon", "wed", "fri"]))
      .toEqual({ current: 6, longest: 6 });
  });

  it("breaks the current streak when a scheduled day is missed", () => {
    mockToday("2026-01-23"); // Friday — Wed Jan 14 was missed
    const completions = [
      "2026-01-12", // Mon
      "2026-01-16", // Fri
      "2026-01-19", // Mon
      "2026-01-21", // Wed
      "2026-01-23", // Fri
    ];
    // Run after the missed Wed: Jan 16 + Jan 19 + Jan 21 + Jan 23 = 4
    expect(computeStreaks(completions, ["mon", "wed", "fri"]))
      .toEqual({ current: 4, longest: 4 });
  });

  it("resets the longest streak after a fully-missed scheduled week", () => {
    mockToday("2026-01-23"); // Friday
    const completions = [
      // Week 1 fully completed
      "2026-01-05",
      "2026-01-07",
      "2026-01-09",
      // Week 2 (Jan 12–18) entirely missed
      // Week 3 fully completed
      "2026-01-19",
      "2026-01-21",
      "2026-01-23",
    ];
    expect(computeStreaks(completions, ["mon", "wed", "fri"]))
      .toEqual({ current: 3, longest: 3 });
  });

  it("keeps the streak alive when today is a non-scheduled day", () => {
    mockToday("2026-01-20"); // Tuesday — not a scheduled day
    const completions = ["2026-01-19"]; // Monday done; Tuesday not scheduled
    expect(computeStreaks(completions, ["mon", "wed", "fri"]))
      .toEqual({ current: 1, longest: 1 });
  });

  it("deduplicates completions when scheduled days are provided", () => {
    mockToday("2026-01-23");
    const completions = [
      "2026-01-21",
      "2026-01-21",
      "2026-01-23",
      "2026-01-23",
    ];
    expect(computeStreaks(completions, ["mon", "wed", "fri"]))
      .toEqual({ current: 2, longest: 2 });
  });

  it("passing all 7 weekdays matches the original calendar-day behavior", () => {
    mockToday("2026-05-25");
    const completions = ["2026-05-24", "2026-05-25"];
    const allDays = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
    expect(computeStreaks(completions, allDays))
      .toEqual({ current: 2, longest: 2 });
  });
});
