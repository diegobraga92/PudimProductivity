import { describe, expect, it, beforeEach } from "vitest";
import {
  DEFAULT_SYNC_SOUND,
  POMODORO_SYNC_ENABLED_KEY,
  POMODORO_SYNC_SOUND_KEY,
  getPomodoroSyncEnabled,
  getPomodoroSyncSound,
  resolveSyncAction,
  setPomodoroSyncEnabled,
  setPomodoroSyncSound,
  subscribe,
} from "./pomodoroSoundSync";
import type { SessionStatus } from "../api/pomodoro";

// Vitest runs in a node environment where `localStorage` is unavailable, so
// provide a minimal in-memory implementation for the store tests.
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

describe("pomodoroSoundSync store", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("defaults to disabled with the white-noise sound", () => {
    expect(getPomodoroSyncEnabled()).toBe(false);
    expect(getPomodoroSyncSound()).toBe(DEFAULT_SYNC_SOUND);
  });

  it("round-trips enabled and sound through localStorage", () => {
    setPomodoroSyncEnabled(true);
    setPomodoroSyncSound("rain");

    expect(localStorage.getItem(POMODORO_SYNC_ENABLED_KEY)).toBe("true");
    expect(localStorage.getItem(POMODORO_SYNC_SOUND_KEY)).toBe("rain");
    expect(getPomodoroSyncEnabled()).toBe(true);
    expect(getPomodoroSyncSound()).toBe("rain");
  });

  it("falls back to the default sound for unknown stored values", () => {
    localStorage.setItem(POMODORO_SYNC_SOUND_KEY, "not-a-sound");
    expect(getPomodoroSyncSound()).toBe(DEFAULT_SYNC_SOUND);
  });

  it("notifies subscribers on change", () => {
    let calls = 0;
    const unsubscribe = subscribe(() => {
      calls += 1;
    });

    setPomodoroSyncEnabled(true);
    setPomodoroSyncSound("ocean");
    unsubscribe();
    setPomodoroSyncEnabled(false);

    expect(calls).toBe(2);
  });
});

describe("resolveSyncAction", () => {
  it.each<[boolean, SessionStatus | null, string]>([
    [false, "running", "idle"],
    [false, null, "idle"],
    [false, "paused", "idle"],
    [true, "running", "play"],
    [true, "paused", "stop"],
    [true, "completed", "stop"],
    [true, "cancelled", "stop"],
    [true, null, "stop"],
  ])("enabled=%s status=%s → %s", (enabled, status, expected) => {
    expect(resolveSyncAction(enabled, status)).toBe(expected);
  });
});
