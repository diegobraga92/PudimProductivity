import { useState, useEffect } from "react";
import { getFeature } from "../api/features";

/**
 * Hook to check if a feature flag is enabled.
 * Returns `false` by default while loading or on error.
 *
 * @param name - The feature flag name to check.
 * @returns `true` if the feature is enabled, `false` otherwise.
 */
export function useFeatureFlag(name: string): boolean {
  const [enabled, setEnabled] = useState(false);

  useEffect(() => {
    let cancelled = false;

    getFeature(name)
      .then((flag) => {
        if (!cancelled) {
          setEnabled(flag.enabled);
        }
      })
      .catch(() => {
        // Feature not found or error — default to disabled
        if (!cancelled) {
          setEnabled(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [name]);

  return enabled;
}