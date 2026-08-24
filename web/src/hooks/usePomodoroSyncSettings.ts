import { useCallback, useSyncExternalStore } from "react";
import type { SoundID } from "../utils/audio";
import {
  getPomodoroSyncEnabled,
  getPomodoroSyncSound,
  setPomodoroSyncEnabled,
  setPomodoroSyncSound,
  subscribe,
} from "../utils/pomodoroSoundSync";

export interface PomodoroSyncSettings {
  /** Whether a sound auto-plays while the pomodoro timer runs. */
  enabled: boolean;
  /** The sound to auto-play. */
  sound: SoundID;
  setEnabled: (enabled: boolean) => void;
  setSound: (sound: SoundID) => void;
}

/**
 * Reactive access to the Pomodoro ↔ Soundscape sync settings. Keeps every
 * consumer (the Pomodoro page's inline control and the app-root automation
 * hook) in sync through one shared store.
 */
export function usePomodoroSyncSettings(): PomodoroSyncSettings {
  const enabled = useSyncExternalStore(subscribe, getPomodoroSyncEnabled);
  const sound = useSyncExternalStore(subscribe, getPomodoroSyncSound);

  const setEnabled = useCallback((v: boolean) => setPomodoroSyncEnabled(v), []);
  const setSound = useCallback((id: SoundID) => setPomodoroSyncSound(id), []);

  return { enabled, sound, setEnabled, setSound };
}
