import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { listScheduledTasks, type Task } from "../api/tasks";
import { useAlarm } from "../components/useAlarm";
import { useI18n } from "../i18n";
import { playAlarmSound } from "../utils/sounds";
import { getToday, parseTimeToMinutes, sanitizeTime } from "../utils/dates";

const CHECK_INTERVAL_MS = 15_000;
const FIRED_ALARMS_KEY = "pudim_fired_alarms";
const DAY_INDEX_TO_KEY = ["sun", "mon", "tue", "wed", "thu", "fri", "sat"] as const;

function isTodayRecurrenceDay(task: Task): boolean {
  const todayKey = DAY_INDEX_TO_KEY[new Date().getDay()];
  return (task.recurrence_days ?? []).includes(todayKey);
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

/** Polls scheduled habits and fires an alarm toast when start_time - alarm_minutes arrives. */
export function useAlarmNotifier(): void {
  const queryClient = useQueryClient();
  const { fireAlarm } = useAlarm();
  const { t } = useI18n();
  const firedRef = useRef<Set<string>>(new Set());
  // Ref keeps the polling effect stable across t() identity changes; written in an effect, not on render.
  const tRef = useRef(t);
  useEffect(() => {
    tRef.current = t;
  }, [t]);

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
          // Write back to the query cache so the planner and later checks reuse the data.
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
          if (lastCheckMinutes !== null) {
            const passedThisCheck = alarmMinutes > lastCheckMinutes && alarmMinutes <= nowMinutes;
            if (!passedThisCheck) continue;
          } else {
            // First check: fire if the alarm is at this minute or within the last 2 minutes.
            if (nowMinutes - alarmMinutes > 2 || nowMinutes - alarmMinutes < 0) continue;
          }

          const alarmKey = `${task.id}:${todayKey}:${task.alarm_minutes}`;
          if (firedRef.current.has(alarmKey)) continue;

          firedRef.current.add(alarmKey);
          saveFiredAlarms(firedRef.current);

          if (isMounted) {
            console.debug(`[alarm] Firing alarm for "${task.title}" (${task.id}) at ${nowMinutes} min, alarm was ${alarmMinutes} min`);
            await playAlarmSound();
            fireAlarm(task);
            // Native OS notification when running inside the Electron desktop app.
            window.desktop?.notify?.({
              title: task.title,
              body: task.start_time ? tRef.current("alarm.desktopTime", { time: sanitizeTime(task.start_time) }) : undefined,
            });
            firedAny = true;
          }
        }

        console.debug(
          `[alarm] check at ${nowMinutes}min (lastCheck=${lastCheckMinutes}) — ${tasks.length} scheduled task(s), fired ${firedAny ? "yes" : "no"}`
        );

        lastCheckMinutes = nowMinutes;
      } catch (err) {
        // API errors.
        console.debug("[alarm] check failed", err);
      }
    };

    checkAlarms();
    const interval = setInterval(checkAlarms, CHECK_INTERVAL_MS);

    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, [queryClient, fireAlarm]);
}