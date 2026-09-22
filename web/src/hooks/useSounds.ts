import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useI18n } from "../i18n";
import { setValidSoundIds } from "../utils/pomodoroSoundSync";
import { resolveSounds, type ResolvedSound } from "../utils/soundCatalog";
import { fetchSoundCatalog, setSoundFileMap } from "../utils/soundFiles";

/**
 * React Query key of the backend sound catalog. Mutations that add, rename or
 * remove a sound invalidate this key so every consumer refreshes.
 */
export const SOUNDS_QUERY_KEY = ["sounds", "catalog"] as const;

/**
 * The soundscape library: built-in sounds (translated labels + emoji) merged
 * with any sounds added by the user, which carry their own stored label/icon.
 *
 * Fetches the backend catalog once and shares it through React Query, so the
 * Soundscape page, the Pomodoro picker and the app-root automation hook all see
 * the same list.
 */
export function useSounds(): { sounds: ResolvedSound[]; isLoading: boolean } {
  const { t } = useI18n();
  const query = useQuery({
    queryKey: SOUNDS_QUERY_KEY,
    queryFn: fetchSoundCatalog,
    // The catalog only changes when a sound is added/removed.
    staleTime: Infinity,
  });

  const sounds = useMemo(() => resolveSounds(query.data ?? [], t), [query.data, t]);

  // The audio engine and the persisted Pomodoro setting are synchronous, so
  // mirror the fetched catalog into them once it arrives.
  useEffect(() => {
    if (!query.data) return;
    setSoundFileMap(query.data);
    setValidSoundIds(query.data.map((entry) => entry.id));
  }, [query.data]);

  return { sounds, isLoading: query.isLoading };
}
