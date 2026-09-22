import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useI18n } from "../i18n";
import { setValidSoundIds } from "../utils/pomodoroSoundSync";
import type { SoundID } from "../utils/audio";
import { SOUNDS } from "../utils/soundCatalog";
import { fetchSoundCatalog, setSoundFileMap } from "../utils/soundFiles";

/** A sound ready to render: resolved label/icon plus where it came from. */
export interface Sound {
  id: SoundID;
  /** Translated name (built-ins) or the user-supplied name (added sounds). */
  label: string;
  icon: string;
  /** True for sounds the user added, rather than shipped with the app. */
  custom: boolean;
}

/**
 * The soundscape library: built-in sounds (translated labels + emoji) merged
 * with any sounds added by the user, which carry their own stored label/icon.
 *
 * Fetches the backend catalog once and shares it through React Query, so the
 * Soundscape page, the Pomodoro picker and the app-root automation hook all see
 * the same list.
 */
export function useSounds(): { sounds: Sound[]; isLoading: boolean } {
  const { t } = useI18n();
  const query = useQuery({
    queryKey: ["sounds", "catalog"],
    queryFn: fetchSoundCatalog,
    // The catalog only changes when a sound is added/removed.
    staleTime: Infinity,
  });

  const sounds = useMemo<Sound[]>(() => {
    const builtinIds = new Set<SoundID>(SOUNDS.map((s) => s.id));
    // Built-ins come first (in catalog order) so their order stays stable even
    // when the backend is unreachable.
    const resolved: Sound[] = SOUNDS.map((s) => ({
      id: s.id,
      label: t(s.labelKey),
      icon: s.icon,
      custom: false,
    }));
    for (const entry of query.data ?? []) {
      if (builtinIds.has(entry.id)) continue;
      resolved.push({
        id: entry.id,
        label: entry.label || entry.id,
        icon: entry.icon || "🎵",
        custom: entry.custom ?? true,
      });
    }
    return resolved;
  }, [query.data, t]);

  // The audio engine and the persisted Pomodoro setting are synchronous, so
  // mirror the fetched catalog into them once it arrives.
  useEffect(() => {
    if (!query.data) return;
    setSoundFileMap(query.data);
    setValidSoundIds(query.data.map((entry) => entry.id));
  }, [query.data]);

  return { sounds, isLoading: query.isLoading };
}
