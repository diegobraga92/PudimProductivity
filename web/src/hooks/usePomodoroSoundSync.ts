import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { getCurrentSession, type SessionStatus } from "../api/pomodoro";
import { getSoundscape } from "../utils/audio";
import { loadSoundCatalog } from "../utils/soundFiles";
import { resolveSyncAction } from "../utils/pomodoroSoundSync";
import { usePomodoroSyncSettings } from "./usePomodoroSyncSettings";

/**
 * App-root pomodoro → sound automation.
 *
 * Mounted once in App.tsx so it outlives the Pomodoro page: the selected sound
 * keeps playing while the timer runs even after the user navigates to another
 * tab, and stops when the timer is paused, completed, cancelled or stopped.
 * The sound only stops if the sync owns it — other manually-played layers in
 * the Soundscape mixer are never touched.
 */
export function usePomodoroSoundSync(): void {
  const { enabled, sound } = usePomodoroSyncSettings();

  // Shares the same cache key as the Pomodoro page, so a start/pause/stop there
  // (which invalidates ["pomodoro"]) refetches promptly here too.
  const { data } = useQuery({
    queryKey: ["pomodoro", "current"],
    queryFn: getCurrentSession,
    refetchInterval: 30_000,
  });

  const status: SessionStatus | null = data?.active ? data.session.status : null;

  // Prime the backend sound file catalog once so the looped MP3s are used when
  // available (the engine synthesizes them otherwise).
  useEffect(() => {
    void loadSoundCatalog();
  }, []);

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
