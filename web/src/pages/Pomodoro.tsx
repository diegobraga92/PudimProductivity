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

const FOCUS_PRESETS = [15, 25, 30, 45, 60];
const BREAK_PRESETS = [5, 10, 15];

function formatTime(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`;
}

function Pomodoro() {
  const queryClient = useQueryClient();
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
      stopMutate.mutate();
    }
  }, [localRemaining, localStatus, stopMutate]);

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
        <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700 }}>🍅 Pomodoro Timer</h2>
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
        {/* Timer Display */}
        <div
          style={{
            fontSize: "4rem",
            fontWeight: 700,
            fontVariantNumeric: "tabular-nums",
            letterSpacing: "2px",
            marginBottom: "var(--space-md)",
            color: isRunning
              ? "var(--color-primary)"
              : isPaused
              ? "var(--color-warning)"
              : "var(--color-text-secondary)",
          }}
        >
          {formatTime(displayTime)}
        </div>

        {/* Progress Bar */}
        <div
          style={{
            width: "100%",
            height: "8px",
            background: "var(--color-bg-secondary)",
            borderRadius: "4px",
            overflow: "hidden",
            marginBottom: "var(--space-lg)",
          }}
        >
          <div
            style={{
              width: `${progress}%`,
              height: "100%",
              background: isRunning
                ? "var(--color-primary)"
                : "var(--color-text-secondary)",
              borderRadius: "4px",
              transition: "width 1s linear",
            }}
          />
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
            ? "Loading..."
            : isRunning
            ? "🔴 Focus time"
            : isPaused
            ? "⏸️ Paused"
            : session?.status === "completed"
            ? "✅ Session completed!"
            : session?.status === "cancelled"
            ? "⏹️ Session cancelled"
            : "Ready to start"}
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
              {startMutate.isPending ? "⏳" : "▶️ Start"}
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
              {pauseMutate.isPending ? "⏳" : "⏸️ Pause"}
            </button>
          )}

          {isPaused && (
            <button
              className="btn btn-primary"
              onClick={() => resumeMutate.mutate()}
              disabled={resumeMutate.isPending}
              style={{ minWidth: "120px" }}
            >
              {resumeMutate.isPending ? "⏳" : "▶️ Resume"}
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
              {stopMutate.isPending ? "⏳" : "⏹️ Stop"}
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
            ⚙️ Settings
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
                Focus (min)
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
                Break (min)
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
      {startMutate.error && (
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
          {startMutate.error.message}
        </div>
      )}
    </div>
  );
}

export default Pomodoro;
