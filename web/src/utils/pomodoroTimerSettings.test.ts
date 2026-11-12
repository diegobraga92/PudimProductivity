import { describe, expect, it, beforeEach } from "vitest";
import {
  BREAK_PRESETS,
  DEFAULT_BREAK_MINUTES,
  DEFAULT_FOCUS_MINUTES,
  FOCUS_PRESETS,
  POMODORO_BREAK_MINUTES_KEY,
  POMODORO_CONTINUOUS_KEY,
  POMODORO_FOCUS_MINUTES_KEY,
  getPomodoroBreakMinutes,
  getPomodoroContinuous,
  getPomodoroFocusMinutes,
  setPomodoroBreakMinutes,
  setPomodoroContinuous,
  setPomodoroFocusMinutes,
  subscribe,
} from "./pomodoroTimerSettings";

// Node has no localStorage, so stub a minimal in-memory one for the store tests.
const storage = new Map<string, string>();
Object.defineProperty(globalThis, "localStorage", {
  configurable: true,
  value: {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => {
      storage.set(key, String(value));
    },
    removeItem: (key: string) => {
      storage.delete(key);
    },
    clear: () => {
      storage.clear();
    },
  },
});

describe("pomodoroTimerSettings store", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("defaults to 25 focus / 5 break minutes and single-shot mode", () => {
    expect(getPomodoroFocusMinutes()).toBe(DEFAULT_FOCUS_MINUTES);
    expect(getPomodoroBreakMinutes()).toBe(DEFAULT_BREAK_MINUTES);
    expect(getPomodoroContinuous()).toBe(false);
  });

  it("round-trips focus and break minutes through localStorage", () => {
    setPomodoroFocusMinutes(90);
    setPomodoroBreakMinutes(20);

    expect(localStorage.getItem(POMODORO_FOCUS_MINUTES_KEY)).toBe("90");
    expect(localStorage.getItem(POMODORO_BREAK_MINUTES_KEY)).toBe("20");
    expect(getPomodoroFocusMinutes()).toBe(90);
    expect(getPomodoroBreakMinutes()).toBe(20);
  });

  it("round-trips continuous mode through the legacy key", () => {
    setPomodoroContinuous(true);

    expect(localStorage.getItem(POMODORO_CONTINUOUS_KEY)).toBe("true");
    expect(getPomodoroContinuous()).toBe(true);
  });

  it("honours a continuous preference stored by older versions", () => {
    localStorage.setItem(POMODORO_CONTINUOUS_KEY, "true");
    expect(getPomodoroContinuous()).toBe(true);
  });

  it.each([
    ["", DEFAULT_FOCUS_MINUTES],
    ["abc", DEFAULT_FOCUS_MINUTES],
    ["-5", DEFAULT_FOCUS_MINUTES],
    ["999", DEFAULT_FOCUS_MINUTES],
    ["27", DEFAULT_FOCUS_MINUTES],
  ])("falls back to the default focus duration for %j", (stored, expected) => {
    localStorage.setItem(POMODORO_FOCUS_MINUTES_KEY, stored);
    expect(getPomodoroFocusMinutes()).toBe(expected);
  });

  it.each([
    ["", DEFAULT_BREAK_MINUTES],
    ["abc", DEFAULT_BREAK_MINUTES],
    ["0", DEFAULT_BREAK_MINUTES],
    ["60", DEFAULT_BREAK_MINUTES],
  ])("falls back to the default break duration for %j", (stored, expected) => {
    localStorage.setItem(POMODORO_BREAK_MINUTES_KEY, stored);
    expect(getPomodoroBreakMinutes()).toBe(expected);
  });

  it("keeps every preset readable", () => {
    for (const minutes of FOCUS_PRESETS) {
      setPomodoroFocusMinutes(minutes);
      expect(getPomodoroFocusMinutes()).toBe(minutes);
    }
    for (const minutes of BREAK_PRESETS) {
      setPomodoroBreakMinutes(minutes);
      expect(getPomodoroBreakMinutes()).toBe(minutes);
    }
  });

  it("notifies subscribers on change", () => {
    let calls = 0;
    const unsubscribe = subscribe(() => {
      calls += 1;
    });

    setPomodoroFocusMinutes(90);
    setPomodoroBreakMinutes(20);
    setPomodoroContinuous(true);
    unsubscribe();
    setPomodoroFocusMinutes(25);

    expect(calls).toBe(3);
  });
});
