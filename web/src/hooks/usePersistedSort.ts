import { useState, useCallback } from "react";
import { VALID_SORT_OPTIONS, type SortOption } from "../utils/sort";

/**
 * Persists a sort option to localStorage and returns the current value
 * plus an update function.
 */
export function usePersistedSort(key: string, initial: SortOption) {
  const [sort, setSort] = useState<SortOption>(() => {
    try {
      const stored = localStorage.getItem(key);
      return stored !== null && VALID_SORT_OPTIONS.has(stored)
        ? (stored as SortOption)
        : initial;
    } catch {
      return initial;
    }
  });

  const updateSort = useCallback(
    (option: SortOption) => {
      try {
        localStorage.setItem(key, option);
      } catch {
        // Ignore storage errors (e.g. private mode)
      }
      setSort(option);
    },
    [key]
  );

  return [sort, updateSort] as const;
}