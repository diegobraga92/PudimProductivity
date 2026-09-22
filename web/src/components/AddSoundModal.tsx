import { useEffect, useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { MAX_SOUND_BYTES, SOUND_FILE_ACCEPT, createSound, updateSound } from "../api/sounds";
import { SOUNDS_QUERY_KEY } from "../hooks/useSounds";
import type { ResolvedSound } from "../utils/soundCatalog";
import Modal from "./Modal";
import { useToast } from "./toastContext";
import { useI18n } from "../i18n";

/** Emoji offered as one-tap picks next to the free-text icon field. */
const ICON_PICKS = ["🌧️", "⛈️", "🔥", "🌊", "🎵", "🌙", "☕", "🚂"];

/** Formats a byte count as megabytes with one decimal place. */
function formatSize(bytes: number): string {
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

interface AddSoundModalProps {
  /** When set, the modal renames/re-icons this sound instead of adding one. */
  sound?: ResolvedSound;
  onClose: () => void;
}

/**
 * Adds a sound to the library (name, icon and audio file) or edits the name and
 * icon of a sound the user already added. The file is checked and previewed
 * before sending, but the backend remains the authority on what it accepts.
 */
export default function AddSoundModal({ sound, onClose }: AddSoundModalProps) {
  const { t } = useI18n();
  const { pushToast } = useToast();
  const queryClient = useQueryClient();

  const isEdit = !!sound;
  const [label, setLabel] = useState(sound?.label ?? "");
  const [icon, setIcon] = useState(sound?.icon ?? "🎵");
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Preview the picked file; the object URL is released when it changes.
  const previewUrl = useMemo(() => (file ? URL.createObjectURL(file) : null), [file]);
  useEffect(
    () => () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl);
    },
    [previewUrl],
  );

  const mutation = useMutation({
    mutationFn: async () => {
      const name = label.trim();
      if (!name) throw new Error(t("soundscape.nameRequired"));
      if (sound) return updateSound(sound.id, { label: name, icon });
      if (!file) throw new Error(t("soundscape.fileRequired"));
      return createSound({ label: name, icon, file });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: SOUNDS_QUERY_KEY });
      pushToast({
        icon: "🎵",
        title: t(isEdit ? "soundscape.soundUpdated" : "soundscape.soundAdded", {
          name: label.trim(),
        }),
      });
      onClose();
    },
    onError: (err: unknown) => {
      setError(err instanceof Error ? err.message : t("soundscape.addFailed"));
    },
  });

  const handleFile = (picked: File | undefined) => {
    setError(null);
    if (!picked) {
      setFile(null);
      return;
    }
    if (picked.size > MAX_SOUND_BYTES) {
      setError(t("soundscape.fileTooLarge"));
      setFile(null);
      return;
    }
    setFile(picked);
  };

  return (
    <Modal onClose={onClose}>
      <h3 className="modal-title">
        {t(isEdit ? "soundscape.editSoundTitle" : "soundscape.addSoundTitle")}
      </h3>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          setError(null);
          mutation.mutate();
        }}
      >
        <div style={{ marginBottom: "var(--space-md)" }}>
          <label className="form-label" htmlFor="sound-name">
            {t("soundscape.soundName")}
          </label>
          <input
            id="sound-name"
            className="input"
            type="text"
            maxLength={60}
            autoFocus
            value={label}
            placeholder={t("soundscape.soundNamePlaceholder")}
            onChange={(e) => setLabel(e.target.value)}
          />
        </div>

        <div style={{ marginBottom: "var(--space-md)" }}>
          <label className="form-label">{t("soundscape.soundIcon")}</label>
          <div style={{ display: "flex", alignItems: "center", gap: "0.35rem", flexWrap: "wrap" }}>
            <input
              className="input"
              type="text"
              maxLength={8}
              value={icon}
              onChange={(e) => setIcon(e.target.value)}
              style={{ width: "4.5rem", textAlign: "center" }}
            />
            {ICON_PICKS.map((pick) => (
              <button
                key={pick}
                type="button"
                className="btn btn-ghost"
                onClick={() => setIcon(pick)}
                style={{ padding: "0.2rem 0.4rem", fontSize: "var(--font-size-base)" }}
              >
                {pick}
              </button>
            ))}
          </div>
        </div>

        {!isEdit && (
          <div style={{ marginBottom: "var(--space-md)" }}>
            <label className="form-label" htmlFor="sound-file">
              {t("soundscape.soundFile")}
            </label>
            <input
              id="sound-file"
              className="input"
              type="file"
              accept={SOUND_FILE_ACCEPT}
              onChange={(e) => handleFile(e.target.files?.[0])}
            />
            <div
              style={{
                fontSize: "var(--font-size-xs)",
                color: "var(--color-text-muted)",
                marginTop: "0.25rem",
              }}
            >
              {t("soundscape.fileHint")}
            </div>
            {file && (
              <div style={{ marginTop: "var(--space-sm)" }}>
                <div style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
                  {file.name} · {formatSize(file.size)}
                </div>
                {previewUrl && (
                  <audio controls src={previewUrl} style={{ width: "100%", marginTop: "0.25rem" }} />
                )}
              </div>
            )}
          </div>
        )}

        {error && (
          <div className="form-error-banner" role="alert">
            {error}
          </div>
        )}

        <div className="modal-actions">
          <button type="button" className="btn btn-ghost" onClick={onClose}>
            {t("common.cancel")}
          </button>
          <button type="submit" className="btn btn-primary" disabled={mutation.isPending}>
            {mutation.isPending ? t("common.saving") : t(isEdit ? "common.save" : "common.add")}
          </button>
        </div>
      </form>
    </Modal>
  );
}

