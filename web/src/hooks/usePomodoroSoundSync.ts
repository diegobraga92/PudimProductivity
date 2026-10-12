import { useEffect } from "react";
import { getSoundscape } from "../utils/audio";
import { resolveSyncAction } from "../utils/pomodoroSoundSync";
import { useCurrentPomodoroStatus } from "./useCurrentPomodoroStatus";
import { useSounds } from "./useSounds";
import { usePomodoroSyncSettings } from "./usePomodoroSyncSettings";

/** App-root pomodoro -> sound automation; keeps playing while the timer runs and stops on any other status. */
export function usePomodoroSoundSync(): void {
  const { enabled, sound } = usePomodoroSyncSettings();
  const status = useCurrentPomodoroStatus();

  // Prime the file catalog so looped files play; also refreshes the ids the persisted setting may reference.
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

