import { useCallback, useSyncExternalStore } from "react";
import {
  getPomodoroBreakMinutes,
  getPomodoroContinuous,
  getPomodoroFocusMinutes,
  setPomodoroBreakMinutes,
  setPomodoroContinuous,
  setPomodoroFocusMinutes,
  subscribe,
} from "../utils/pomodoroTimerSettings";

export interface PomodoroTimerSettings {
  /** Focus segment length in minutes. */
  focusMinutes: number;
  /** Break segment length in minutes. */
  breakMinutes: number;
  /** Whether the timer auto-advances focus ↔ break. */
  continuous: boolean;
  setFocusMinutes: (minutes: number) => void;
  setBreakMinutes: (minutes: number) => void;
  setContinuous: (enabled: boolean) => void;
}

/** Reactive pomodoro timer settings, persisted across reloads and page switches. */
export function usePomodoroTimerSettings(): PomodoroTimerSettings {
  const focusMinutes = useSyncExternalStore(subscribe, getPomodoroFocusMinutes);
  const breakMinutes = useSyncExternalStore(subscribe, getPomodoroBreakMinutes);
  const continuous = useSyncExternalStore(subscribe, getPomodoroContinuous);

  const setFocusMinutes = useCallback((v: number) => setPomodoroFocusMinutes(v), []);
  const setBreakMinutes = useCallback((v: number) => setPomodoroBreakMinutes(v), []);
  const setContinuous = useCallback((v: boolean) => setPomodoroContinuous(v), []);

  return {
    focusMinutes,
    breakMinutes,
    continuous,
    setFocusMinutes,
    setBreakMinutes,
    setContinuous,
  };
}
