import { useEffect, useRef } from "react";
import type { SessionStatus } from "../api/pomodoro";
import { useToast } from "../components/toastContext";
import { useI18n } from "../i18n";
import { resolveMediaAction } from "../utils/pomodoroMediaSync";
import { useCurrentPomodoroStatus } from "./useCurrentPomodoroStatus";
import { usePomodoroMediaSettings } from "./usePomodoroMediaSettings";
import { useSystemMediaPlayers } from "./useSystemMediaPlayers";

/** App-root pomodoro -> system player automation; recomputed only when the status string changes. */
export function usePomodoroMediaSync(): void {
  const { enabled, player } = usePomodoroMediaSettings();
  const { availability } = useSystemMediaPlayers();
  const { pushToast } = useToast();
  const { t } = useI18n();
  const warnedRef = useRef(false);
  // undefined means the timer status was never observed yet; the first
  // observation must not touch the OS player (opening the app used to pause it).
  const previousStatusRef = useRef<SessionStatus | null | undefined>(undefined);
  const status = useCurrentPomodoroStatus();
  const available = availability === "available";

  // A fresh detection re-arms the warning, so a later removal is reported once.
  useEffect(() => {
    if (available) warnedRef.current = false;
  }, [available]);

  useEffect(() => {
    const previousStatus = previousStatusRef.current;
    previousStatusRef.current = status;

    if (!enabled || !available) return;

    const bridge = window.desktop;
    if (!bridge?.mediaControl) return;

    const action = resolveMediaAction(enabled, available, previousStatus, status);
    if (action === "idle") return;

    void bridge
      .mediaControl({ action, player: player || undefined })
      .then((result) => {
        if (result.ok || result.reason !== "unavailable" || warnedRef.current) return;
        warnedRef.current = true;
        pushToast({
          icon: "⚠️",
          title: t("pomodoro.mediaMissing"),
          body: t("pomodoro.mediaMissingHint"),
        });
      })
      .catch(() => {
        // Bridge failures must stay invisible to the timer.
      });
  }, [enabled, available, player, status, pushToast, t]);
}
