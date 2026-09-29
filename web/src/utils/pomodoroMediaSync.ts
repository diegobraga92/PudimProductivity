import type { SessionStatus } from "../api/pomodoro";

/**
 * Pomodoro to system media player sync settings.
 *
 * Stores the enabled flag and the chosen player in localStorage, and exposes a
 * small external store so the Pomodoro page and the app root hook stay in sync.
 */

export const POMODORO_MEDIA_ENABLED_KEY = "soundscape_pomodoro_media_enabled";
export const POMODORO_MEDIA_PLAYER_KEY = "soundscape_pomodoro_media_player";

/** Empty string lets playerctl choose, which follows the last active player. */
export const AUTO_PLAYER = "";

export function getPomodoroMediaEnabled(): boolean {
  return localStorage.getItem(POMODORO_MEDIA_ENABLED_KEY) === "true";
}

export function setPomodoroMediaEnabled(enabled: boolean): void {
  localStorage.setItem(POMODORO_MEDIA_ENABLED_KEY, String(enabled));
  emit();
}

export function getPomodoroMediaPlayer(): string {
  return localStorage.getItem(POMODORO_MEDIA_PLAYER_KEY) ?? AUTO_PLAYER;
}

export function setPomodoroMediaPlayer(player: string): void {
  localStorage.setItem(POMODORO_MEDIA_PLAYER_KEY, player);
  emit();
}

/** Detection state of the desktop media bridge. */
export type MediaAvailability = "checking" | "available" | "unsupported" | "missing";

type Listener = () => void;
const listeners = new Set<Listener>();

/** Subscribes to setting changes. Returns an unsubscribe function. */
export function subscribe(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function emit(): void {
  for (const listener of listeners) listener();
}

export type MediaSyncAction = "play" | "pause" | "idle";

/**
 * Decides what the automation hook should do for a given timer status.
 * Disabled sync and an unusable playerctl never touch the system player, so
 * the feature stays inert instead of reporting errors.
 */
export function resolveMediaAction(
  enabled: boolean,
  available: boolean,
  status: SessionStatus | null
): MediaSyncAction {
  if (!enabled || !available) return "idle";
  return status === "running" ? "play" : "pause";
}
