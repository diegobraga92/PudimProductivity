/**
 * Minimal external store, the `subscribe` half of React's
 * `useSyncExternalStore`. Shared by the localStorage-backed settings modules so
 * they don't each re-implement listener bookkeeping.
 */
export interface ExternalStore {
  /** Subscribes to store changes. Returns an unsubscribe function. */
  subscribe(listener: () => void): () => void;
  /** Notifies every subscriber that the store changed. */
  emit(): void;
}

export function createExternalStore(): ExternalStore {
  const listeners = new Set<() => void>();
  return {
    subscribe(listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    emit() {
      for (const listener of listeners) listener();
    },
  };
}
