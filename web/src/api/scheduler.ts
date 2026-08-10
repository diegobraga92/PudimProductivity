import config from "../config";
import type { components } from "./generated/scheduler-v1";

// Types are generated from api/openapi/scheduler-v1.yaml.
export type Suggestion = components["schemas"]["Suggestion"];
export type ScheduleSlot = components["schemas"]["ScheduleSlot"];

export async function getDailySchedule(date?: string): Promise<Suggestion> {
  const q = date ? `?date=${date}` : "";
  const res = await fetch(`${config.apiBaseUrl}/schedule${q}`);
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error || `Failed to get schedule: ${res.status}`);
  }
  return res.json() as Promise<Suggestion>;
}
