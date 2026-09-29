import { beforeEach, describe, expect, it } from "vitest";
import type { SessionStatus } from "../api/pomodoro";
import {
  AUTO_PLAYER,
  POMODORO_MEDIA_ENABLED_KEY,
  POMODORO_MEDIA_PLAYER_KEY,
  getPomodoroMediaEnabled,
  getPomodoroMediaPlayer,
  resolveMediaAction,
  setPomodoroMediaEnabled,
  setPomodoroMediaPlayer,
  subscribe,
} from "./pomodoroMediaSync";

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

describe("pomodoroMediaSync store", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("defaults to disabled with the automatic player", () => {
    expect(getPomodoroMediaEnabled()).toBe(false);
    expect(getPomodoroMediaPlayer()).toBe(AUTO_PLAYER);
  });

  it("round-trips enabled and player through localStorage", () => {
    setPomodoroMediaEnabled(true);
    setPomodoroMediaPlayer("spotify");

    expect(localStorage.getItem(POMODORO_MEDIA_ENABLED_KEY)).toBe("true");
    expect(localStorage.getItem(POMODORO_MEDIA_PLAYER_KEY)).toBe("spotify");
    expect(getPomodoroMediaEnabled()).toBe(true);
    expect(getPomodoroMediaPlayer()).toBe("spotify");
  });

  it("translates any non true value into disabled", () => {
    localStorage.setItem(POMODORO_MEDIA_ENABLED_KEY, "yes");
    expect(getPomodoroMediaEnabled()).toBe(false);
  });

  it("notifies subscribers on change", () => {
    let calls = 0;
    const unsubscribe = subscribe(() => {
      calls += 1;
    });

    setPomodoroMediaEnabled(true);
    setPomodoroMediaPlayer("spotify");
    unsubscribe();
    setPomodoroMediaEnabled(false);

    expect(calls).toBe(2);
  });
});

describe("resolveMediaAction", () => {
  it.each<[boolean, boolean, SessionStatus | null, string]>([
    [false, true, "running", "idle"],
    [false, true, null, "idle"],
    [true, false, "running", "idle"],
    [true, false, "paused", "idle"],
    [false, false, null, "idle"],
    [true, true, "running", "play"],
    [true, true, "paused", "pause"],
    [true, true, "completed", "pause"],
    [true, true, "cancelled", "pause"],
    [true, true, null, "pause"],
  ])("enabled=%s available=%s status=%s returns %s", (enabled, available, status, expected) => {
    expect(resolveMediaAction(enabled, available, status)).toBe(expected);
  });
});
