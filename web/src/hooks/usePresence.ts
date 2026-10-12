import { useCallback, useEffect, useState } from "react";
import { getListPresence } from "../api/collab";
import { syncClient, type WsEvent } from "../api/sync";

/** Online users per list: REST snapshot + presence.online/offline events -> Map<listId, Set<userId>>. */
export function usePresence(listIds: string[]): {
  online: Map<string, Set<string>>;
  isOnline: (listId: string, userId: string) => boolean;
  refresh: (listId: string) => Promise<void>;
} {
  const [online, setOnline] = useState<Map<string, Set<string>>>(() => new Map());

  const apply = useCallback((listId: string, users: string[]) => {
    setOnline((prev) => {
      const next = new Map(prev);
      next.set(listId, new Set(users));
      return next;
    });
  }, []);

  const addUser = useCallback((listId: string, userId: string) => {
    setOnline((prev) => {
      const next = new Map(prev);
      const users = new Set(next.get(listId) ?? []);
      users.add(userId);
      next.set(listId, users);
      return next;
    });
  }, []);

  // online events only fire on connect, so a later page load bootstraps from the REST snapshot.
  const refresh = useCallback(
    async (listId: string) => {
      try {
        const res = await getListPresence(listId);
        apply(listId, res.online);
      } catch {
        // Backend may be degraded; presence is best-effort.
      }
    },
    [apply]
  );

  useEffect(() => {
    listIds.forEach((id) => void refresh(id));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [listIds.join(",")]);

  useEffect(() => {
    const onPresence = (event: WsEvent) => {
      const payload = (event.payload ?? {}) as {
        user_id?: string;
        list_ids?: string[];
      };
      const userId = payload.user_id;
      if (!userId) return;

      if (event.type === "presence.online") {
        const lists = payload.list_ids ?? [];
        if (lists.length === 0) {
          // No shared lists; remove from any locally tracked list.
          setOnline((prev) => {
            const next = new Map(prev);
            next.forEach((users, listId) => {
              const u = new Set(users);
              u.delete(userId);
              next.set(listId, u);
            });
            return next;
          });
        } else {
          lists.forEach((listId) => addUser(listId, userId));
        }
      } else if (event.type === "presence.offline") {
        setOnline((prev) => {
          const next = new Map(prev);
          next.forEach((users, listId) => {
            const u = new Set(users);
            u.delete(userId);
            next.set(listId, u);
          });
          return next;
        });
      }
    };

    const unsubscribe = syncClient.on("presence.online", onPresence);
    const unsubscribeOffline = syncClient.on("presence.offline", onPresence);
    return () => {
      unsubscribe();
      unsubscribeOffline();
    };
  }, [addUser]);

  const isOnline = useCallback(
    (listId: string, userId: string) => online.get(listId)?.has(userId) ?? false,
    [online]
  );

  return { online, isOnline, refresh };
}
