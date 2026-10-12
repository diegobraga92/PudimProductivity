import { useQuery } from "@tanstack/react-query";
import { getCurrentSession, type SessionStatus } from "../api/pomodoro";
import { queryKeys } from "../utils/queryKeys";

/**
 * Reads the current pomodoro session status, polling every 30s, and returns null
 * when no session is active. Shares one React Query cache entry with the
 * Pomodoro page, so a start/pause/stop there is reflected here promptly.
 */
export function useCurrentPomodoroStatus(): SessionStatus | null {
  const { data } = useQuery({
    queryKey: queryKeys.pomodoroCurrent,
    queryFn: getCurrentSession,
    refetchInterval: 30_000,
  });

  return data?.active ? data.session.status : null;
}
