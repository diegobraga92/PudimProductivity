import { useEffect, useRef, useState } from "react";
import { getSoundscape, type SoundID, type PresetID } from "../utils/audio";
import { loadSoundCatalog } from "../utils/soundFiles";
import { useI18n } from "../i18n";
import { SOUNDS } from "../utils/soundCatalog";
import { MusicIcon } from "../components/icons";

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
        maxWidth: "560px",
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
    "light-rain": 0.5,
    rain: 0.5,
    "rain-and-thunder": 0.5,
    "strong-rain": 0.5,
    "stronger-rain": 0.5,
    fire: 0.5,
    "fire-and-thunder": 0.5,
    ocean: 0.5,
  });
  const [presets, setPresets] = useState<{ id: PresetID; label: string }[]>([]);

  const soundscape = getSoundscape();

  // Fetch the backend sound file catalog once so sound buttons play the real
  // audio loops served by the backend.
  useEffect(() => {
    void loadSoundCatalog();
  }, []);

  // Load presets on mount
  useEffect(() => {
    setPresets(soundscape.getPresets().map((p) => ({ id: p.id, label: p.label })));
  }, [soundscape]);

  // Sync master volume
  useEffect(() => {
    soundscape.setVolume(masterVolume);
  }, [masterVolume, soundscape]);

  const toggle = (id: SoundID) => {
    if (playing.has(id)) {
      // Stop with fade-out
      soundscape.stop(id);
      setPlaying((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    } else if (soundscape.play(id)) {
      soundscape.setSoundVolume(id, volumes[id]);
      setPlaying((prev) => new Set(prev).add(id));
    }
  };

  const changeVolume = (id: SoundID, v: number) => {
    setVolumes((prev) => ({ ...prev, [id]: v }));
    soundscape.setSoundVolume(id, v);
  };

  /** Save current mix as a preset. */
  const savePreset = () => {
    const label = prompt(t("soundscape.presetPrompt"));
    if (!label) return;
    const currentSounds: Partial<Record<SoundID, boolean>> = {};
    for (const id of SOUNDS.map((s) => s.id)) {
      currentSounds[id] = playing.has(id);
    }
    soundscape.savePreset(label, currentSounds, volumes, masterVolume);
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
      if (preset.sounds[id] && soundscape.play(id)) {
        const vol = preset.volumes[id] ?? 0.5;
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
  };

  /** Delete a preset. */
  const deletePreset = (presetId: PresetID) => {
    soundscape.deletePreset(presetId);
    setPresets(soundscape.getPresets().map((p) => ({ id: p.id, label: p.label })));
  };

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
        <h2 className="page-heading" style={{ marginBottom: 0 }}><MusicIcon size={24} /> {t("soundscape.title")}</h2>
      </div>

      {/* Frequency Visualizer */}
      <Visualizer />

      {/* Master Volume */}
      <div
        className="card"
        style={{
          maxWidth: "560px",
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
        <button
          className="btn btn-ghost"
          onClick={() => {
            soundscape.stopAll();
            setPlaying(new Set());
          }}
          style={{ fontSize: "var(--font-size-xs)", whiteSpace: "nowrap" }}
          title={t("soundscape.stopAll")}
        >
          {t("soundscape.stopAll")}
        </button>
      </div>

      {/* Sound Cards */}
      <div
        style={{
          maxWidth: "560px",
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
                <div
                  style={{
                    fontWeight: 600,
                    fontSize: "var(--font-size-sm)",
                    whiteSpace: "nowrap",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                  }}
                >
                  {sound.icon} {t(sound.labelKey)}
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

      {/* Presets */}
      <div
        className="card"
        style={{
          maxWidth: "560px",
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

      {/* Tip */}
      <div
        style={{
          maxWidth: "560px",
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