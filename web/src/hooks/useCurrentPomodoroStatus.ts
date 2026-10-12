import { useQuery } from "@tanstack/react-query";
import { getCurrentSession, type SessionStatus } from "../api/pomodoro";
import { queryKeys } from "../utils/queryKeys";

/** Current session status, polled every 30s (null when idle); shares the Pomodoro page's cache entry. */
export function useCurrentPomodoroStatus(): SessionStatus | null {
  const { data } = useQuery({
    queryKey: queryKeys.pomodoroCurrent,
    queryFn: getCurrentSession,
    refetchInterval: 30_000,
  });

  return data?.active ? data.session.status : null;
}
