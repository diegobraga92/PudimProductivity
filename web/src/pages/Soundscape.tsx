import { useEffect, useRef, useState } from "react";
import { getSoundscape, type SoundID, type PresetID } from "../utils/audio";
import { useI18n } from "../i18n";

interface SoundDef {
  id: SoundID;
  labelKey: string;
  icon: string;
  descKey: string;
}

const SOUNDS: SoundDef[] = [
  { id: "white-noise", labelKey: "soundscape.whiteNoise", icon: "🌊", descKey: "soundscape.desc.whiteNoise" },
  { id: "pink-noise", labelKey: "soundscape.pinkNoise", icon: "🌸", descKey: "soundscape.desc.pinkNoise" },
  { id: "brown-noise", labelKey: "soundscape.brownNoise", icon: "🌫️", descKey: "soundscape.desc.brownNoise" },
  { id: "rain", labelKey: "soundscape.rainSound", icon: "🌧️", descKey: "soundscape.desc.rain" },
  { id: "ocean", labelKey: "soundscape.ocean", icon: "🌊", descKey: "soundscape.desc.ocean" },
  { id: "wind", labelKey: "soundscape.wind", icon: "💨", descKey: "soundscape.desc.wind" },
  { id: "campfire", labelKey: "soundscape.campfire", icon: "🔥", descKey: "soundscape.desc.campfire" },
  { id: "binaural-beat", labelKey: "soundscape.binaural", icon: "🎧", descKey: "soundscape.desc.binaural" },
  { id: "isochronic-tone", labelKey: "soundscape.isochronic", icon: "📳", descKey: "soundscape.desc.isochronic" },
  { id: "meditation-bowl", labelKey: "soundscape.meditationBowl", icon: "🕉️", descKey: "soundscape.desc.meditationBowl" },
  { id: "ambient-pad", labelKey: "soundscape.ambientPad", icon: "🎹", descKey: "soundscape.desc.ambientPad" },
];

const LS_POMODORO_ENABLED = "soundscape_pomodoro_enabled";
const LS_POMODORO_SOUND = "soundscape_pomodoro_sound";
const LS_RAIN_INTENSITY = "soundscape_rain_intensity";

// Rain intensity translation keys
const RAIN_LABEL_KEYS = [
  "soundscape.rainDrizzle",
  "soundscape.rainLight",
  "soundscape.rainModerate",
  "soundscape.rainHeavy",
  "soundscape.rainDownpour",
];

/** Canvas-based frequency visualizer. */
function Visualizer() {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const animRef = useRef<number>(0);

  useEffect(() => {
    const soundscape = getSoundscape();
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const width = canvas.width;
    const height = canvas.height;
    const data = new Uint8Array(128);

    const draw = () => {
      soundscape.getFrequencyData(data);
      ctx.clearRect(0, 0, width, height);

      const barCount = 64;
      const barWidth = width / barCount;

      for (let i = 0; i < barCount; i++) {
        const value = data[i] / 255;
        const barHeight = value * height;

        // Gradient: blue → cyan → green → yellow
        const hue = 200 - value * 120;
        ctx.fillStyle = `hsl(${hue}, 80%, ${50 + value * 30}%)`;
        ctx.fillRect(i * barWidth, height - barHeight, barWidth - 1, barHeight);
      }

      animRef.current = requestAnimationFrame(draw);
    };

    draw();

    return () => {
      cancelAnimationFrame(animRef.current);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      width={480}
      height={80}
      style={{
        width: "100%",
        maxWidth: "480px",
        height: "80px",
        borderRadius: "8px",
        display: "block",
        margin: "0 auto var(--space-md)",
        background: "rgba(0,0,0,0.05)",
      }}
    />
  );
}

function Soundscape() {
  const { t } = useI18n();
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
    "binaural-beat": 0.5,
    "isochronic-tone": 0.5,
    "meditation-bowl": 0.5,
    "ambient-pad": 0.5,
  });
  const [rainIntensity, setRainIntensity] = useState(() => {
    const saved = localStorage.getItem(LS_RAIN_INTENSITY);
    return saved ? parseFloat(saved) : 0.5;
  });
  const [presets, setPresets] = useState<{ id: PresetID; label: string }[]>([]);

  // Pomodoro sync state
  const [pomodoroEnabled, setPomodoroEnabled] = useState(() => {
    return localStorage.getItem(LS_POMODORO_ENABLED) === "true";
  });
  const [pomodoroSound, setPomodoroSound] = useState<SoundID>(() => {
    return (localStorage.getItem(LS_POMODORO_SOUND) as SoundID) || "white-noise";
  });

  const soundscape = getSoundscape();

  // Load presets on mount
  useEffect(() => {
    setPresets(soundscape.getPresets().map((p) => ({ id: p.id, label: p.label })));
  }, [soundscape]);

  // Sync master volume
  useEffect(() => {
    soundscape.setVolume(masterVolume);
  }, [masterVolume, soundscape]);

  // Sync rain intensity
  useEffect(() => {
    soundscape.setRainIntensity(rainIntensity);
  }, [rainIntensity, soundscape]);

  const toggle = (id: SoundID) => {
    if (playing.has(id)) {
      // Stop with fade-out
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

  const changeRainIntensity = (v: number) => {
    setRainIntensity(v);
    localStorage.setItem(LS_RAIN_INTENSITY, String(v));
  };

  /** Save current mix as a preset. */
  const savePreset = () => {
    const label = prompt(t("soundscape.presetPrompt"));
    if (!label) return;
    const currentSounds: Partial<Record<SoundID, boolean>> = {};
    for (const id of SOUNDS.map((s) => s.id)) {
      currentSounds[id] = playing.has(id);
    }
    soundscape.savePreset(label, currentSounds, volumes, masterVolume, rainIntensity);
    setPresets(soundscape.getPresets().map((p) => ({ id: p.id, label: p.label })));
  };

  /** Load a preset. */
  const loadPreset = (presetId: PresetID) => {
    const allPresets = soundscape.getPresets();
    const preset = allPresets.find((p) => p.id === presetId);
    if (!preset) return;

    // Stop all current sounds
    for (const id of playing) {
      soundscape.stop(id, false);
    }
    setPlaying(new Set());

    // Apply preset
    const newPlaying = new Set<SoundID>();
    for (const id of SOUNDS.map((s) => s.id)) {
      if (preset.sounds[id]) {
        const vol = preset.volumes[id] ?? 0.5;
        soundscape.play(id);
        soundscape.setSoundVolume(id, vol);
        newPlaying.add(id);
      }
    }

    // Restore volumes
    const newVolumes = { ...volumes };
    for (const id of SOUNDS.map((s) => s.id)) {
      if (preset.volumes[id] !== undefined) {
        newVolumes[id] = preset.volumes[id];
      }
    }
    setVolumes(newVolumes);
    setPlaying(newPlaying);
    setMasterVolume(preset.masterVolume);
    setRainIntensity(preset.rainIntensity);
  };

  /** Delete a preset. */
  const deletePreset = (presetId: PresetID) => {
    soundscape.deletePreset(presetId);
    setPresets(soundscape.getPresets().map((p) => ({ id: p.id, label: p.label })));
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
        <h2 className="page-heading" style={{ marginBottom: 0 }}>{t("soundscape.title")}</h2>
      </div>

      {/* Frequency Visualizer */}
      <Visualizer />

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
          🔊 {t("soundscape.master")}
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
                {isOn ? t("soundscape.pause") : t("soundscape.play")}
              </button>

              {/* Info */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontWeight: 600, fontSize: "var(--font-size-sm)" }}>
                  {sound.icon} {t(sound.labelKey)}
                </div>
                <div style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
                  {t(sound.descKey)}
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

      {/* Rain Intensity Slider — visible when rain is playing */}
      {playing.has("rain") && (
        <div
          className="card"
          style={{
            maxWidth: "480px",
            margin: "var(--space-md) auto 0",
            padding: "var(--space-md) var(--space-lg)",
          }}
        >
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: "var(--space-md)",
            }}
          >
            <span style={{ fontSize: "var(--font-size-sm)", fontWeight: 600, minWidth: "60px" }}>
              🌧️ {t("soundscape.rain")}
            </span>
            <input
              type="range"
              min="0"
              max="1"
              step="0.01"
              value={rainIntensity}
              onChange={(e) => changeRainIntensity(parseFloat(e.target.value))}
              style={{ flex: 1, accentColor: "var(--color-primary)" }}
            />
            <span style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)", minWidth: "80px", textAlign: "right" }}>
              {t(RAIN_LABEL_KEYS[Math.round(rainIntensity * 4)])}
            </span>
          </div>
        </div>
      )}

      {/* Presets */}
      <div
        className="card"
        style={{
          maxWidth: "480px",
          margin: "var(--space-md) auto 0",
          padding: "var(--space-md) var(--space-lg)",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            marginBottom: "var(--space-sm)",
          }}
        >
          <div>
            <div style={{ fontWeight: 600, fontSize: "var(--font-size-sm)" }}>
              {t("soundscape.presets")}
            </div>
          </div>
          <button className="btn btn-primary" onClick={savePreset} style={{ fontSize: "var(--font-size-xs)" }}>
            {t("soundscape.saveCurrent")}
          </button>
        </div>

        {presets.length === 0 && (
          <div style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-muted)", textAlign: "center", padding: "var(--space-sm)" }}>
            {t("soundscape.noPresets")}
          </div>
        )}

        {presets.map((p) => (
          <div
            key={p.id}
            style={{
              display: "flex",
              alignItems: "center",
              gap: "var(--space-sm)",
              padding: "var(--space-xs) 0",
            }}
          >
            <button
              className="btn btn-ghost"
              onClick={() => loadPreset(p.id)}
              style={{
                flex: 1,
                textAlign: "left",
                fontSize: "var(--font-size-sm)",
                padding: "var(--space-xs) var(--space-sm)",
              }}
            >
              🎵 {p.label}
            </button>
            <button
              className="btn btn-ghost"
              onClick={() => deletePreset(p.id)}
              style={{
                fontSize: "var(--font-size-xs)",
                color: "var(--color-danger, #e74c3c)",
                padding: "var(--space-xs)",
              }}
              title={t("soundscape.deletePreset")}
            >
              ✕
            </button>
          </div>
        ))}
      </div>

      {/* Pomodoro Sync */}
      <div
        className="card"
        style={{
          maxWidth: "480px",
          margin: "var(--space-md) auto 0",
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
              {t("soundscape.pomodoroSync")}
            </div>
            <div style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
              {t("soundscape.pomodoroSyncDesc")}
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
              {t("soundscape.sound")}
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
                  {s.icon} {t(s.labelKey)}
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
        💡 {t("soundscape.tip")}
      </div>
    </div>
  );
}

export default Soundscape;