import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { getCurrentSession, type SessionStatus } from "../api/pomodoro";
import { useToast } from "../components/toastContext";
import { useI18n } from "../i18n";
import { resolveMediaAction } from "../utils/pomodoroMediaSync";
import { usePomodoroMediaSettings } from "./usePomodoroMediaSettings";
import { useSystemMediaPlayers } from "./useSystemMediaPlayers";

/**
 * App root pomodoro to system media player automation.
 *
 * Mounted once in App.tsx so it outlives the Pomodoro page: the external player
 * resumes while the timer runs and pauses on any other status, even after the
 * user navigates to another tab. The action is recomputed only when the status
 * string actually changes, so the 30s poll does not reissue commands.
 */
export function usePomodoroMediaSync(): void {
  const { enabled, player } = usePomodoroMediaSettings();
  const { availability } = useSystemMediaPlayers();
  const { pushToast } = useToast();
  const { t } = useI18n();
  const warnedRef = useRef(false);

  // Shares the same cache key as the Pomodoro page, so a start or stop there
  // refetches promptly here too.
  const { data } = useQuery({
    queryKey: ["pomodoro", "current"],
    queryFn: getCurrentSession,
    refetchInterval: 30_000,
  });

  const status: SessionStatus | null = data?.active ? data.session.status : null;
  const available = availability === "available";

  // A fresh detection re-arms the warning, so a later removal is reported once.
  useEffect(() => {
    if (available) warnedRef.current = false;
  }, [available]);

  useEffect(() => {
    if (!enabled || !available) return;

    const bridge = window.desktop;
    if (!bridge?.mediaControl) return;

    const action = resolveMediaAction(enabled, available, status);
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
