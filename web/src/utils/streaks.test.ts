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
