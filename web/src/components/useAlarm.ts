import { createContext, useContext } from "react";
import type { Task } from "../api/tasks";

export interface ActiveAlarm {
  id: string;
  task: Task;
  firedAt: string; // ISO timestamp
}

export interface AlarmContextValue {
  activeAlarms: ActiveAlarm[];
  fireAlarm: (task: Task) => void;
  dismissAlarm: (alarmId: string) => void;
}

export const AlarmContext = createContext<AlarmContextValue | null>(null);

export function useAlarm(): AlarmContextValue {
  const ctx = useContext(AlarmContext);
  if (!ctx) {
    throw new Error("useAlarm must be used within AlarmProvider");
  }
  return ctx;
}