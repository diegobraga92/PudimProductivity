import { useCallback, useMemo, useRef, useState, type ReactNode } from "react";
import type { Task } from "../api/tasks";
import { AlarmContext, type ActiveAlarm, type AlarmContextValue } from "./useAlarm";

export function AlarmProvider({ children }: { children: ReactNode }) {
  const [activeAlarms, setActiveAlarms] = useState<ActiveAlarm[]>([]);
  const idCounter = useRef(0);

  const fireAlarm = useCallback((task: Task) => {
    idCounter.current += 1;
    const alarm: ActiveAlarm = {
      id: `alarm-${idCounter.current}`,
      task,
      firedAt: new Date().toISOString(),
    };
    setActiveAlarms((prev) => [alarm, ...prev]);
  }, []);

  const dismissAlarm = useCallback((alarmId: string) => {
    setActiveAlarms((prev) => prev.filter((a) => a.id !== alarmId));
  }, []);

  const value = useMemo<AlarmContextValue>(
    () => ({ activeAlarms, fireAlarm, dismissAlarm }),
    [activeAlarms, fireAlarm, dismissAlarm]
  );

  return <AlarmContext.Provider value={value}>{children}</AlarmContext.Provider>;
}