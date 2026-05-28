import { useEffect, useState } from "react";
import { getSoundscape, type SoundID } from "../utils/audio";

interface SoundDef {
  id: SoundID;
  label: string;
  icon: string;
  description: string;
}

const SOUNDS: SoundDef[] = [
  { id: "white-noise", label: "White Noise", icon: "🌊", description: "Flat static — crisp and neutral" },
  { id: "pink-noise", label: "Pink Noise", icon: "🌸", description: "Softer highs — warmer, more natural" },
  { id: "brown-noise", label: "Brown Noise", icon: "🌫️", description: "Deep rumble — calming bass" },
  { id: "rain", label: "Rain", icon: "🌧️", description: "Gentle rainfall with movement" },
  { id: "ocean", label: "Ocean", icon: "🌊", description: "Slow waves with a natural swell" },
  { id: "wind", label: "Wind", icon: "💨", description: "Howling wind with occasional gusts" },
  { id: "campfire", label: "Campfire", icon: "🔥", description: "Crackling fire with embers and pops" },
];

const LS_POMODORO_ENABLED = "soundscape_pomodoro_enabled";
const LS_POMODORO_SOUND = "soundscape_pomodoro_sound";

function Soundscape() {
  const [playing, setPlaying] = useState<Set<SoundID>>(new Set());
  const [masterVolume, setMasterVolume] = useState(0.5);
  const [volumes, setVolumes] = useState<Record<SoundID, number>>({
    "white-noise": 0.5,
    "pink-noise": 0.5,
    "brown-noise": 0.5,
    rain: 0.5,
    ocean: 0.5,
    wind: 0.5,
    campfire: 0.5,
  });

  // Pomodoro sync state (persisted to localStorage)
  const [pomodoroEnabled, setPomodoroEnabled] = useState(() => {
    return localStorage.getItem(LS_POMODORO_ENABLED) === "true";
  });
  const [pomodoroSound, setPomodoroSound] = useState<SoundID>(() => {
    return (localStorage.getItem(LS_POMODORO_SOUND) as SoundID) || "white-noise";
  });

  const soundscape = getSoundscape();

  // Sync master volume
  useEffect(() => {
    soundscape.setVolume(masterVolume);
  }, [masterVolume, soundscape]);

  const toggle = (id: SoundID) => {
    if (playing.has(id)) {
      soundscape.stop(id);
      setPlaying((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    } else {
      soundscape.play(id);
      soundscape.setSoundVolume(id, volumes[id]);
      setPlaying((prev) => new Set(prev).add(id));
    }
  };

  const changeVolume = (id: SoundID, v: number) => {
    setVolumes((prev) => ({ ...prev, [id]: v }));
    soundscape.setSoundVolume(id, v);
  };

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      soundscape.stopAll();
    };
  }, [soundscape]);

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
        <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700 }}>🎵 Soundscape</h2>
      </div>

      {/* Master Volume */}
      <div
        className="card"
        style={{
          maxWidth: "480px",
          margin: "0 auto var(--space-md)",
          padding: "var(--space-md) var(--space-lg)",
          display: "flex",
          alignItems: "center",
          gap: "var(--space-md)",
        }}
      >
        <span style={{ fontSize: "var(--font-size-sm)", fontWeight: 600, minWidth: "60px" }}>
          🔊 Master
        </span>
        <input
          type="range"
          min="0"
          max="1"
          step="0.01"
          value={masterVolume}
          onChange={(e) => setMasterVolume(parseFloat(e.target.value))}
          style={{ flex: 1, accentColor: "var(--color-primary)" }}
        />
        <span style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)", minWidth: "32px", textAlign: "right" }}>
          {Math.round(masterVolume * 100)}%
        </span>
      </div>

      {/* Sound Cards */}
      <div
        style={{
          maxWidth: "480px",
          margin: "0 auto",
          display: "flex",
          flexDirection: "column",
          gap: "var(--space-sm)",
        }}
      >
        {SOUNDS.map((sound) => {
          const isOn = playing.has(sound.id);
          return (
            <div
              key={sound.id}
              className="card"
              style={{
                padding: "var(--space-md) var(--space-lg)",
                display: "flex",
                alignItems: "center",
                gap: "var(--space-md)",
                borderColor: isOn ? "var(--color-primary)" : undefined,
                boxShadow: isOn ? "0 0 0 2px var(--color-primary-light)" : undefined,
                transition: "all var(--transition-fast)",
              }}
            >
              {/* Play/Pause Button */}
              <button
                className={`btn ${isOn ? "btn-primary" : "btn-ghost"}`}
                onClick={() => toggle(sound.id)}
                style={{
                  minWidth: "80px",
                  fontSize: "var(--font-size-base)",
                }}
              >
                {isOn ? "⏸️ Pause" : "▶️ Play"}
              </button>

              {/* Info */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontWeight: 600, fontSize: "var(--font-size-sm)" }}>
                  {sound.icon} {sound.label}
                </div>
                <div style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
                  {sound.description}
                </div>
              </div>

              {/* Volume Slider */}
              <div style={{ display: "flex", alignItems: "center", gap: "0.35rem", minWidth: "100px" }}>
                <span style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-muted)" }}>🔉</span>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.01"
                  value={volumes[sound.id]}
                  onChange={(e) => changeVolume(sound.id, parseFloat(e.target.value))}
                  style={{ flex: 1, accentColor: "var(--color-primary)" }}
                />
              </div>
            </div>
          );
        })}
      </div>

      {/* Pomodoro Sync */}
      <div
        className="card"
        style={{
          maxWidth: "480px",
          margin: "var(--space-lg) auto 0",
          padding: "var(--space-md) var(--space-lg)",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            marginBottom: pomodoroEnabled ? "var(--space-sm)" : 0,
          }}
        >
          <div>
            <div style={{ fontWeight: 600, fontSize: "var(--font-size-sm)" }}>
              🎯 Pomodoro Sync
            </div>
            <div style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
              Auto-play sound while timer is running
            </div>
          </div>
          <label
            style={{
              position: "relative",
              display: "inline-block",
              width: "44px",
              height: "24px",
              cursor: "pointer",
            }}
          >
            <input
              type="checkbox"
              checked={pomodoroEnabled}
              onChange={(e) => {
                const v = e.target.checked;
                setPomodoroEnabled(v);
                localStorage.setItem(LS_POMODORO_ENABLED, String(v));
              }}
              style={{ display: "none" }}
            />
            <span
              style={{
                position: "absolute",
                inset: 0,
                borderRadius: "24px",
                background: pomodoroEnabled ? "var(--color-primary)" : "var(--color-border)",
                transition: "background var(--transition-fast)",
              }}
            >
              <span
                style={{
                  position: "absolute",
                  top: "2px",
                  left: pomodoroEnabled ? "22px" : "2px",
                  width: "20px",
                  height: "20px",
                  borderRadius: "50%",
                  background: "white",
                  transition: "left var(--transition-fast)",
                  boxShadow: "0 1px 3px rgba(0,0,0,0.2)",
                }}
              />
            </span>
          </label>
        </div>

        {pomodoroEnabled && (
          <div style={{ display: "flex", alignItems: "center", gap: "var(--space-sm)", marginTop: "var(--space-sm)" }}>
            <span style={{ fontSize: "var(--font-size-sm)", fontWeight: 500, whiteSpace: "nowrap" }}>
              Sound:
            </span>
            <select
              className="select"
              value={pomodoroSound}
              onChange={(e) => {
                const v = e.target.value as SoundID;
                setPomodoroSound(v);
                localStorage.setItem(LS_POMODORO_SOUND, v);
              }}
              style={{ flex: 1 }}
            >
              {SOUNDS.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.icon} {s.label}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      {/* Tip */}
      <div
        style={{
          maxWidth: "480px",
          margin: "var(--space-md) auto 0",
          textAlign: "center",
          fontSize: "var(--font-size-xs)",
          color: "var(--color-text-muted)",
        }}
      >
        💡 You can layer multiple sounds at once. Adjust each volume to create your perfect mix.
      </div>
    </div>
  );
}

export default Soundscape;
