import type { SessionStatus } from "../api/pomodoro";
import { createExternalStore } from "./externalStore";

/** Pomodoro -> system player sync settings (enabled + player) in localStorage, with an external store. */

export const POMODORO_MEDIA_ENABLED_KEY = "soundscape_pomodoro_media_enabled";
export const POMODORO_MEDIA_PLAYER_KEY = "soundscape_pomodoro_media_player";

/** Empty string lets playerctl choose, which follows the last active player. */
export const AUTO_PLAYER = "";

const store = createExternalStore();

export function getPomodoroMediaEnabled(): boolean {
  return localStorage.getItem(POMODORO_MEDIA_ENABLED_KEY) === "true";
}

export function setPomodoroMediaEnabled(enabled: boolean): void {
  localStorage.setItem(POMODORO_MEDIA_ENABLED_KEY, String(enabled));
  store.emit();
}

export function getPomodoroMediaPlayer(): string {
  return localStorage.getItem(POMODORO_MEDIA_PLAYER_KEY) ?? AUTO_PLAYER;
}

export function setPomodoroMediaPlayer(player: string): void {
  localStorage.setItem(POMODORO_MEDIA_PLAYER_KEY, player);
  store.emit();
}

/** Detection state of the desktop media bridge. */
export type MediaAvailability = "checking" | "available" | "unsupported" | "missing";

/** Subscribes to setting changes. Returns an unsubscribe function. */
export const subscribe = store.subscribe;

export type MediaSyncAction = "play" | "pause" | "idle";

/**
 * Action for one status observation; disabled sync and an unusable playerctl
 * stay inert instead of erroring.
 *
 * Only real transitions act. The first observation (`previousStatus` undefined)
 * is inert, so launching the app never touches the media that is already
 * playing in the OS, and pause only fires when a running timer stops.
 */
export function resolveMediaAction(
  enabled: boolean,
  available: boolean,
  previousStatus: SessionStatus | null | undefined,
  status: SessionStatus | null
): MediaSyncAction {
  if (!enabled || !available) return "idle";
  if (previousStatus === undefined) return "idle";
  if (status === "running" && previousStatus !== "running") return "play";
  if (previousStatus === "running" && status !== "running") return "pause";
  return "idle";
}
