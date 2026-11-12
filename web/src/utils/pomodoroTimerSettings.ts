import { createExternalStore } from "./externalStore";

/** Pomodoro timer settings (focus/break minutes + continuous mode) in localStorage, with an external store. */

export const POMODORO_FOCUS_MINUTES_KEY = "pomodoro_focus_minutes";
export const POMODORO_BREAK_MINUTES_KEY = "pomodoro_break_minutes";
/** Legacy key kept so existing installs keep their continuous preference. */
export const POMODORO_CONTINUOUS_KEY = "pomodoro_continuous";

export const FOCUS_PRESETS = [15, 25, 30, 45, 60, 90, 120];
export const BREAK_PRESETS = [5, 10, 15, 20, 25, 30];
export const DEFAULT_FOCUS_MINUTES = 25;
export const DEFAULT_BREAK_MINUTES = 5;

const store = createExternalStore();

/** Stored minutes validated against the presets; missing or unknown values fall back. */
function readMinutes(key: string, presets: readonly number[], fallback: number): number {
  try {
    const raw = localStorage.getItem(key);
    if (raw === null) return fallback;
    const minutes = Number(raw);
    return Number.isInteger(minutes) && presets.includes(minutes) ? minutes : fallback;
  } catch {
    return fallback;
  }
}

function writeMinutes(key: string, minutes: number): void {
  try {
    localStorage.setItem(key, String(minutes));
  } catch {
    // Ignore storage errors (e.g. private mode)
  }
  store.emit();
}

export function getPomodoroFocusMinutes(): number {
  return readMinutes(POMODORO_FOCUS_MINUTES_KEY, FOCUS_PRESETS, DEFAULT_FOCUS_MINUTES);
}

export function setPomodoroFocusMinutes(minutes: number): void {
  writeMinutes(POMODORO_FOCUS_MINUTES_KEY, minutes);
}

export function getPomodoroBreakMinutes(): number {
  return readMinutes(POMODORO_BREAK_MINUTES_KEY, BREAK_PRESETS, DEFAULT_BREAK_MINUTES);
}

export function setPomodoroBreakMinutes(minutes: number): void {
  writeMinutes(POMODORO_BREAK_MINUTES_KEY, minutes);
}

export function getPomodoroContinuous(): boolean {
  try {
    return localStorage.getItem(POMODORO_CONTINUOUS_KEY) === "true";
  } catch {
    return false;
  }
}

export function setPomodoroContinuous(enabled: boolean): void {
  try {
    localStorage.setItem(POMODORO_CONTINUOUS_KEY, String(enabled));
  } catch {
    // Ignore storage errors (e.g. private mode)
  }
  store.emit();
}

/** Subscribes to timer-setting changes. Returns an unsubscribe function. */
export const subscribe = store.subscribe;
