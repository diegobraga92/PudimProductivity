import { useCallback, useEffect, useState } from "react";
import type { MediaAvailability } from "../utils/pomodoroMediaSync";

export interface SystemMediaPlayers {
  /** Detection state, see MediaAvailability. */
  availability: MediaAvailability;
  /** Player names reported by playerctl. */
  players: string[];
  /** Runs detection again, used after the user installs playerctl. */
  recheck: () => void;
}

/**
 * Detects whether the desktop shell can control the system media player.
 * The bridge is absent in the browser, which reports the feature as
 * unsupported so the UI can hide it entirely.
 */
export function useSystemMediaPlayers(): SystemMediaPlayers {
  // Without the bridge the feature must be hidden on the very first render,
  // otherwise the browser build flashes a control it can never use.
  const [availability, setAvailability] = useState<MediaAvailability>(() =>
    window.desktop?.mediaStatus ? "checking" : "unsupported"
  );
  const [players, setPlayers] = useState<string[]>([]);

  const check = useCallback(async () => {
    const bridge = window.desktop;

    if (!bridge?.mediaStatus || !bridge.mediaListPlayers) {
      setAvailability("unsupported");
      setPlayers([]);
      return;
    }

    try {
      const status = await bridge.mediaStatus();

      if (!status.supported) {
        setAvailability("unsupported");
        setPlayers([]);
        return;
      }

      if (!status.available) {
        setAvailability("missing");
        setPlayers([]);
        return;
      }

      const list = await bridge.mediaListPlayers();
      setPlayers(list.players);
      setAvailability("available");
    } catch {
      setAvailability("missing");
      setPlayers([]);
    }
  }, []);

  useEffect(() => {
    void check();
  }, [check]);

  const recheck = useCallback(() => {
    setAvailability("checking");
    void check();
  }, [check]);

  return { availability, players, recheck };
}
