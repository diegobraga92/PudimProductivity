import { useEffect } from "react";
import { getSoundscape } from "../utils/audio";
import { resolveSyncAction } from "../utils/pomodoroSoundSync";
import { useCurrentPomodoroStatus } from "./useCurrentPomodoroStatus";
import { useSounds } from "./useSounds";
import { usePomodoroSyncSettings } from "./usePomodoroSyncSettings";

/**
 * App-root pomodoro → sound automation.
 *
 * Mounted once in App.tsx so it outlives the Pomodoro page: the selected sound
 * keeps playing while the timer runs even after the user navigates to another
 * tab, and stops when the timer is paused, completed, cancelled or stopped.
 */
export function usePomodoroSoundSync(): void {
  const { enabled, sound } = usePomodoroSyncSettings();
  const status = useCurrentPomodoroStatus();

  // Prime the backend sound file catalog so the looped audio files are used.
  // The hook also mirrors the catalog into the engine's URL map and refreshes
  // the set of ids the persisted sync setting may reference.
  useSounds();

  useEffect(() => {
    const action = resolveSyncAction(enabled, status);
    const soundscape = getSoundscape();

    if (action === "play") {
      if (!soundscape.isPlaying(sound)) {
        soundscape.play(sound);
      }
    } else if (action === "stop") {
      soundscape.stop(sound);
    }
  }, [enabled, sound, status]);
}

