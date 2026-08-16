import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import {
  getCurrentSession,
  startSession,
  pauseSession,
  resumeSession,
  stopSession,
} from "../api/pomodoro";
import { getSoundscape, type SoundID } from "../utils/audio";
import { useI18n } from "../i18n";

const FOCUS_PRESETS = [15, 25, 30, 45, 60];
const BREAK_PRESETS = [5, 10, 15];
const RING_RADIUS = 96;
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS;

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
}

function Pomodoro() {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [focusMinutes, setFocusMinutes] = useState(25);
  const [breakMinutes, setBreakMinutes] = useState(5);
  const [localRemaining, setLocalRemaining] = useState<number | null>(null);
  const [localStatus, setLocalStatus] = useState<string | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Fetch current session on mount and every 30s
  const { data: currentResp, isLoading } = useQuery({
    queryKey: ["pomodoro", "current"],
    queryFn: getCurrentSession,
    refetchInterval: 30_000,
  });

  const session = currentResp?.active ? currentResp.session : null;

  // Sync local state from server — only on session identity or status change,
  // NOT on every refetch (remaining_seconds changes every poll).
  useEffect(() => {
    if (session) {
      setLocalRemaining(session.remaining_seconds);
      setLocalStatus(session.status);
    } else {
      setLocalRemaining(null);
      setLocalStatus(null);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session?.id, session?.status]);

  const stopMutate = useMutation({
    mutationFn: stopSession,
    onSuccess: () => {
      setLocalRemaining(null);
      setLocalStatus(null);
      queryClient.invalidateQueries({ queryKey: ["pomodoro"] });
    },
  });

  // Use a ref to avoid stale closure issues in the auto-stop effect
  const stopMutateRef = useRef(stopMutate);
  stopMutateRef.current = stopMutate;

  // Local ticking
  useEffect(() => {
    if (localStatus === "running") {
      intervalRef.current = setInterval(() => {
        setLocalRemaining((prev) => (prev !== null && prev > 0 ? prev - 1 : 0));
      }, 1000);
    } else {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [localStatus]);

  // When timer hits 0 while running, auto-stop
  useEffect(() => {
    if (localRemaining === 0 && localStatus === "running") {
      stopMutateRef.current.mutate();
    }
  }, [localRemaining, localStatus]);

  const startMutate = useMutation({
    mutationFn: () =>
      startSession({ focus_duration: focusMinutes, break_duration: breakMinutes }),
    onSuccess: (data) => {
      setLocalRemaining(data.remaining_seconds);
      setLocalStatus(data.status);
      queryClient.invalidateQueries({ queryKey: ["pomodoro"] });
    },
  });

  const pauseMutate = useMutation({
    mutationFn: pauseSession,
    onSuccess: (data) => {
      setLocalRemaining(data.remaining_seconds);
      setLocalStatus(data.status);
      queryClient.invalidateQueries({ queryKey: ["pomodoro"] });
    },
  });

  const resumeMutate = useMutation({
    mutationFn: resumeSession,
    onSuccess: (data) => {
      setLocalRemaining(data.remaining_seconds);
      setLocalStatus(data.status);
      queryClient.invalidateQueries({ queryKey: ["pomodoro"] });
    },
  });

  const isRunning = localStatus === "running";
  const isPaused = localStatus === "paused";
  const isActive = isRunning || isPaused;
  const displayTime = localRemaining !== null ? localRemaining : (session?.remaining_seconds ?? 0);
  const totalSeconds = session ? session.focus_duration * 60 : focusMinutes * 60;
  const progress = totalSeconds > 0 ? ((totalSeconds - displayTime) / totalSeconds) * 100 : 0;

  // ── Pomodoro ↔ Soundscape sync ──
  // Reads the user's preference from localStorage (set in Soundscape page).
  // Plays the selected sound when the timer runs, stops when paused/stopped/completed.
  useEffect(() => {
    const enabled = localStorage.getItem("soundscape_pomodoro_enabled") === "true";
    if (!enabled) return;

    const soundId = (localStorage.getItem("soundscape_pomodoro_sound") || "white-noise") as SoundID;
    const soundscape = getSoundscape();

    if (localStatus === "running") {
      // Only play if not already playing (e.g. from manual toggle)
      if (!soundscape.isPlaying(soundId)) {
        soundscape.play(soundId);
      }
    } else if (localStatus === "paused" || localStatus === "completed" || localStatus === "cancelled" || localStatus === null) {
      soundscape.stop(soundId);
    }
  }, [localStatus]);

  // Stop sound on unmount
  useEffect(() => {
    return () => {
      const enabled = localStorage.getItem("soundscape_pomodoro_enabled") === "true";
      if (!enabled) return;
      const soundId = (localStorage.getItem("soundscape_pomodoro_sound") || "white-noise") as SoundID;
      getSoundscape().stop(soundId);
    };
  }, []);

  // Desktop: prevent the OS from suspending while the focus timer runs so the
  // countdown and audio stay accurate (Electron powerSaveBlocker; no-op in the
  // plain browser).
  useEffect(() => {
    window.desktop?.setPowerSaveBlocker?.(localStatus === "running");
    return () => window.desktop?.setPowerSaveBlocker?.(false);
  }, [localStatus]);

  return (
    <div className="animate-fade-in">
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "var(--space-sm)",
          marginBottom: "var(--space-lg)",
        }}
      >
        <h2 className="page-heading" style={{ marginBottom: 0 }}>{t("pomodoro.title")}</h2>
      </div>

      <div
        className="card"
        style={{
          maxWidth: "480px",
          margin: "0 auto",
          textAlign: "center",
          padding: "var(--space-xl)",
        }}
      >
        {/* Circular Countdown Ring */}
        <div
          style={{
            position: "relative",
            width: 220,
            height: 220,
            margin: "0 auto var(--space-lg)",
          }}
        >
          <svg width="220" height="220" viewBox="0 0 220 220" role="img" aria-label={t("pomodoro.remainingAria", { time: formatTime(displayTime) })}>
            <circle
              cx="110"
              cy="110"
              r="96"
              fill="none"
              stroke="var(--color-border-light)"
              strokeWidth="14"
            />
            <circle
              cx="110"
              cy="110"
              r="96"
              fill="none"
              stroke={
                isRunning
                  ? "var(--color-primary)"
                  : isPaused
                  ? "var(--color-warning)"
                  : "var(--color-text-muted)"
              }
              strokeWidth="14"
              strokeLinecap="round"
              strokeDasharray={RING_CIRCUMFERENCE}
              strokeDashoffset={RING_CIRCUMFERENCE * (1 - progress / 100)}
              transform="rotate(-90 110 110)"
              style={{ transition: "stroke-dashoffset 1s linear, stroke 250ms ease" }}
            />
          </svg>
          <div
            style={{
              position: "absolute",
              inset: 0,
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <span
              style={{
                fontSize: "3.25rem",
                fontWeight: 700,
                fontVariantNumeric: "tabular-nums",
                letterSpacing: "2px",
                lineHeight: 1.1,
                color: isRunning
                  ? "var(--color-primary)"
                  : isPaused
                  ? "var(--color-warning)"
                  : "var(--color-text)",
              }}
            >
              {formatTime(displayTime)}
            </span>
            <span style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)", letterSpacing: "1px" }}>
              {isActive ? t("pomodoro.focus") : t("pomodoro.ready")}
            </span>
          </div>
        </div>

        {/* Status */}
        <div
          style={{
            marginBottom: "var(--space-lg)",
            fontSize: "var(--font-size-sm)",
            color: "var(--color-text-secondary)",
          }}
        >
          {isLoading
            ? t("common.loadingDot")
            : isRunning
            ? t("pomodoro.focusTime")
            : isPaused
            ? t("pomodoro.paused")
            : session?.status === "completed"
            ? t("pomodoro.completed")
            : session?.status === "cancelled"
            ? t("pomodoro.cancelled")
            : t("pomodoro.readyToStart")}
        </div>

        {/* Controls */}
        <div
          style={{
            display: "flex",
            gap: "var(--space-sm)",
            justifyContent: "center",
            flexWrap: "wrap",
          }}
        >
          {!isActive && (
            <button
              className="btn btn-primary"
              onClick={() => startMutate.mutate()}
              disabled={startMutate.isPending}
              style={{ minWidth: "120px" }}
            >
              {startMutate.isPending ? "⏳" : t("pomodoro.start")}
            </button>
          )}

          {isRunning && (
            <button
              className="btn"
              onClick={() => pauseMutate.mutate()}
              disabled={pauseMutate.isPending}
              style={{
                background: "var(--color-warning)",
                color: "white",
                minWidth: "120px",
              }}
            >
              {pauseMutate.isPending ? "⏳" : t("pomodoro.pause")}
            </button>
          )}

          {isPaused && (
            <button
              className="btn btn-primary"
              onClick={() => resumeMutate.mutate()}
              disabled={resumeMutate.isPending}
              style={{ minWidth: "120px" }}
            >
              {resumeMutate.isPending ? "⏳" : t("pomodoro.resume")}
            </button>
          )}

          {isActive && (
            <button
              className="btn"
              onClick={() => stopMutate.mutate()}
              disabled={stopMutate.isPending}
              style={{
                background: "var(--color-danger)",
                color: "white",
                minWidth: "120px",
              }}
            >
              {stopMutate.isPending ? "⏳" : t("pomodoro.stop")}
            </button>
          )}
        </div>
      </div>

      {/* Settings (only when no session is active) */}
      {!isActive && (
        <div
          className="card"
          style={{
            maxWidth: "480px",
            margin: "var(--space-md) auto 0",
            padding: "var(--space-lg)",
          }}
        >
          <h3
            style={{
              fontSize: "var(--font-size-md)",
              fontWeight: 600,
              marginBottom: "var(--space-md)",
            }}
          >
            {t("pomodoro.settings")}
          </h3>

          <div
            style={{
              display: "flex",
              gap: "var(--space-lg)",
              flexWrap: "wrap",
              justifyContent: "center",
            }}
          >
            {/* Focus Duration */}
            <div>
              <label
                style={{
                  display: "block",
                  fontSize: "var(--font-size-sm)",
                  fontWeight: 500,
                  marginBottom: "var(--space-xs)",
                  color: "var(--color-text-secondary)",
                }}
              >
                {t("pomodoro.focusMinutes")}
              </label>
              <div style={{ display: "flex", gap: "0.25rem", flexWrap: "wrap" }}>
                {FOCUS_PRESETS.map((m) => (
                  <button
                    key={m}
                    className={`btn btn-sm ${focusMinutes === m ? "btn-primary" : ""}`}
                    onClick={() => setFocusMinutes(m)}
                    style={{ minWidth: "40px" }}
                  >
                    {m}
                  </button>
                ))}
              </div>
            </div>

            {/* Break Duration */}
            <div>
              <label
                style={{
                  display: "block",
                  fontSize: "var(--font-size-sm)",
                  fontWeight: 500,
                  marginBottom: "var(--space-xs)",
                  color: "var(--color-text-secondary)",
                }}
              >
                {t("pomodoro.breakMinutes")}
              </label>
              <div style={{ display: "flex", gap: "0.25rem", flexWrap: "wrap" }}>
                {BREAK_PRESETS.map((m) => (
                  <button
                    key={m}
                    className={`btn btn-sm ${breakMinutes === m ? "btn-primary" : ""}`}
                    onClick={() => setBreakMinutes(m)}
                    style={{ minWidth: "40px" }}
                  >
                    {m}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Error display */}
      {(startMutate.error || pauseMutate.error || resumeMutate.error || stopMutate.error) && (
        <div
          className="card"
          style={{
            maxWidth: "480px",
            margin: "var(--space-md) auto 0",
            padding: "var(--space-md)",
            background: "var(--color-danger-light, #ffe0e0)",
            color: "var(--color-danger)",
            textAlign: "center",
          }}
        >
          {startMutate.error?.message || pauseMutate.error?.message || resumeMutate.error?.message || stopMutate.error?.message}
        </div>
      )}
    </div>
  );
}

export default Pomodoro;
