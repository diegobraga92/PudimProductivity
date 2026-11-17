import { beforeEach, describe, expect, it } from "vitest";
import type { SessionStatus } from "../api/pomodoro";
import {
  AUTO_PLAYER,
  POMODORO_MEDIA_ENABLED_KEY,
  POMODORO_MEDIA_PLAYER_KEY,
  getPomodoroMediaEnabled,
  getPomodoroMediaPlayer,
  type MediaSyncAction,
  resolveMediaAction,
  setPomodoroMediaEnabled,
  setPomodoroMediaPlayer,
  subscribe,
} from "./pomodoroMediaSync";

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
  it.each<
    [boolean, boolean, SessionStatus | null | undefined, SessionStatus | null, MediaSyncAction]
  >([
    [false, true, null, "running", "idle"],
    [false, true, undefined, null, "idle"],
    [true, false, "running", "paused", "idle"],
    [true, false, undefined, null, "idle"],
    [false, false, undefined, null, "idle"],
    // First observation is launch: never touch the media already playing in the OS.
    [true, true, undefined, null, "idle"],
    [true, true, undefined, "paused", "idle"],
    [true, true, undefined, "completed", "idle"],
    [true, true, undefined, "cancelled", "idle"],
    [true, true, undefined, "running", "idle"],
    // Real transitions.
    [true, true, null, "running", "play"],
    [true, true, "paused", "running", "play"],
    [true, true, "completed", "running", "play"],
    [true, true, "running", "paused", "pause"],
    [true, true, "running", "completed", "pause"],
    [true, true, "running", "cancelled", "pause"],
    [true, true, "running", null, "pause"],
    // Statuses that did not change stay inert (no repeated play/pause calls).
    [true, true, null, null, "idle"],
    [true, true, "running", "running", "idle"],
    [true, true, "paused", "paused", "idle"],
    [true, true, "paused", null, "idle"],
    [true, true, "completed", null, "idle"],
  ])(
    "enabled=%s available=%s previous=%s status=%s returns %s",
    (enabled, available, previousStatus, status, expected) => {
      expect(resolveMediaAction(enabled, available, previousStatus, status)).toBe(expected);
    }
  );

  it("does not pause the system player when the app opens", () => {
    // The reported bug: no active session on launch must not reach playerctl.
    expect(resolveMediaAction(true, true, undefined, null)).toBe("idle");
  });

  it("pauses only after a running session ends", () => {
    const statuses: (SessionStatus | null)[] = [null, "running", "completed", null];
    let previous: SessionStatus | null | undefined = undefined;
    const actions = statuses.map((status) => {
      const action = resolveMediaAction(true, true, previous, status);
      previous = status;
      return action;
    });

    expect(actions).toEqual(["idle", "play", "pause", "idle"]);
  });
});
