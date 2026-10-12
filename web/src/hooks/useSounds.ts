import { useQuery } from "@tanstack/react-query";
import { useEffect, useMemo } from "react";
import { useI18n } from "../i18n";
import { setValidSoundIds } from "../utils/pomodoroSoundSync";
import { resolveSounds, type ResolvedSound } from "../utils/soundCatalog";
import { fetchSoundCatalog, setSoundFileMap } from "../utils/soundFiles";

/** React Query key of the sound catalog; add/rename/remove invalidate it. */
export const SOUNDS_QUERY_KEY = ["sounds", "catalog"] as const;

/** Built-in sounds (translated labels) merged with user-added ones, shared through React Query. */
export function useSounds(): { sounds: ResolvedSound[]; isLoading: boolean } {
  const { t } = useI18n();
  const query = useQuery({
    queryKey: SOUNDS_QUERY_KEY,
    queryFn: fetchSoundCatalog,
    // The catalog only changes when a sound is added/removed.
    staleTime: Infinity,
  });

  const sounds = useMemo(() => resolveSounds(query.data ?? [], t), [query.data, t]);

  // The engine and persisted setting are synchronous, so mirror the catalog into them once fetched.
  useEffect(() => {
    if (!query.data) return;
    setSoundFileMap(query.data);
    setValidSoundIds(query.data.map((entry) => entry.id));
  }, [query.data]);

  return { sounds, isLoading: query.isLoading };
}
