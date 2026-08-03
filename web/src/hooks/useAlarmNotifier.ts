import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { listScheduledTasks, type Task } from "../api/tasks";
import { playAlarmSound } from "../utils/sounds";
import { getToday } from "../utils/dates";

const CHECK_INTERVAL_MS = 15_000;
const FIRED_ALARMS_KEY = "pudim_fired_alarms";
const DAY_INDEX_TO_KEY = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"] as const;

function isTodayRecurrenceDay(task: Task): boolean {
  const todayKey = DAY_INDEX_TO_KEY[new Date().getDay()];
  return (task.recurrence_days ?? []).includes(todayKey);
}

function parseTimeToMinutes(t: string): number {
  const [h, m] = t.split(":").map(Number);
  return h * 60 + (m || 0);
}

function loadFiredAlarms(): Set<string> {
  try {
    const raw = localStorage.getItem(FIRED_ALARMS_KEY);
    if (!raw) return new Set();
    return new Set(JSON.parse(raw) as string[]);
  } catch {
    return new Set();
  }
}

function saveFiredAlarms(keys: Set<string>): void {
  try {
    const today = getToday();
    const filtered = [...keys].filter((k) => k.startsWith(`${today}:`));
    localStorage.setItem(FIRED_ALARMS_KEY, JSON.stringify(filtered));
  } catch {
    // Ignore storage errors
  }
}

async function ensureNotificationPermission(): Promise<boolean> {
  if (!("Notification" in window)) return false;
  if (Notification.permission === "granted") return true;
  if (Notification.permission === "denied") return false;
  const permission = await Notification.requestPermission();
  return permission === "granted";
}

async function showAlarmNotification(task: Task): Promise<void> {
  try {
    if (!(await ensureNotificationPermission())) return;
    new Notification(`⏰ ${task.title}`, {
      body: `${task.title} starts at ${task.start_time} • ${task.alarm_minutes} minute${(task.alarm_minutes ?? 0) > 1 ? "s" : ""} from now!`,
      tag: `alarm-${task.id}`,
    });
  } catch {
    // Notification API unavailable — silently ignore
  }
}

/**
 * Polls scheduled habit tasks and fires an alarm (sound + notification) when
 * the alarm time (start_time minus alarm_minutes) arrives for a habit that
 * recurs on today.
 */
export function useAlarmNotifier(): void {
  const queryClient = useQueryClient();
  const firedRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    firedRef.current = loadFiredAlarms();
  }, []);

  useEffect(() => {
    let isMounted = true;
    let lastCheckMinutes: number | null = null;

    const checkAlarms = async () => {
      try {
        const cached = queryClient.getQueryData<Task[]>(["scheduledTasks"]);
        let tasks: Task[] = cached ?? [];
        if (tasks.length === 0) {
          tasks = await listScheduledTasks();
          // Write back to the react-query cache so the planner and future
          // alarm checks share the same data without redundant fetches.
          queryClient.setQueryData<Task[]>(["scheduledTasks"], tasks);
        }

        const now = new Date();
        const nowMinutes = now.getHours() * 60 + now.getMinutes();
        const todayKey = getToday();

        let firedAny = false;

        for (const task of tasks) {
          if (!task.recurrence_days || !task.start_time || task.alarm_minutes == null || task.alarm_minutes <= 0) {
            continue;
          }
          if (!isTodayRecurrenceDay(task)) {
            continue;
          }

          const startMinutes = parseTimeToMinutes(task.start_time);
          const alarmMinutes = startMinutes - task.alarm_minutes;
          if (alarmMinutes < 0) continue;

          // Fire when the alarm time falls between the previous check and now.
          // This catches alarms even if the app loads slightly after the minute.
          if (lastCheckMinutes !== null) {
            const passedThisCheck = alarmMinutes > lastCheckMinutes && alarmMinutes <= nowMinutes;
            if (!passedThisCheck) continue;
          } else {
            // First check: fire if the alarm is at the current minute
            // or was within the last 2 minutes (grace period for late loads).
            if (nowMinutes - alarmMinutes > 2 || nowMinutes - alarmMinutes < 0) continue;
          }

          const alarmKey = `${task.id}:${todayKey}:${task.alarm_minutes}`;
          if (firedRef.current.has(alarmKey)) continue;

          firedRef.current.add(alarmKey);
          saveFiredAlarms(firedRef.current);

          if (isMounted) {
            console.debug(`[alarm] Firing alarm for "${task.title}" (${task.id}) at ${nowMinutes} min, alarm was ${alarmMinutes} min`);
            await playAlarmSound();
            void showAlarmNotification(task);
            firedAny = true;
          }
        }

        console.debug(
          `[alarm] check at ${nowMinutes}min (lastCheck=${lastCheckMinutes}) — ${tasks.length} scheduled task(s), fired ${firedAny ? "yes" : "no"}`
        );

        lastCheckMinutes = nowMinutes;
      } catch (err) {
        // API errors — log instead of silently swallowing so future
        // alarm issues are diagnosable.
        console.debug("[alarm] check failed", err);
      }
    };

    checkAlarms();
    const interval = setInterval(checkAlarms, CHECK_INTERVAL_MS);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [queryClient]);
}