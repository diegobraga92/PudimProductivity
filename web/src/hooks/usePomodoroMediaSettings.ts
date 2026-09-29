import { useCallback, useSyncExternalStore } from "react";
import {
  getPomodoroMediaEnabled,
  getPomodoroMediaPlayer,
  setPomodoroMediaEnabled,
  setPomodoroMediaPlayer,
  subscribe,
} from "../utils/pomodoroMediaSync";

export interface PomodoroMediaSettings {
  /** Whether the pomodoro timer drives the system media player. */
  enabled: boolean;
  /** Player name to target, or an empty string for the playerctl default. */
  player: string;
  setEnabled: (enabled: boolean) => void;
  setPlayer: (player: string) => void;
}

/** Reactive access to the pomodoro and system media player sync settings. */
export function usePomodoroMediaSettings(): PomodoroMediaSettings {
  const enabled = useSyncExternalStore(subscribe, getPomodoroMediaEnabled);
  const player = useSyncExternalStore(subscribe, getPomodoroMediaPlayer);

  const setEnabled = useCallback((v: boolean) => setPomodoroMediaEnabled(v), []);
  const setPlayer = useCallback((p: string) => setPomodoroMediaPlayer(p), []);

  return { enabled, player, setEnabled, setPlayer };
}
