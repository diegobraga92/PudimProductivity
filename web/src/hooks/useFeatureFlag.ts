import { useState, useEffect } from "react";
import { getFeature } from "../api/features";

/** Feature flag name -> enabled; false while loading or on error. */
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
        // Feature not found or error. Default to disabled
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