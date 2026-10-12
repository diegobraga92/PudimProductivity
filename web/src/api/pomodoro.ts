import { request } from "./client";
import type { components, paths } from "./generated/pomodoro-v1";

// Types are generated from api/openapi/pomodoro-v1.yaml (the source of truth).
export type SessionStatus = components["schemas"]["SessionStatus"];
export type PomodoroSession = components["schemas"]["PomodoroSession"];
export type StartSessionRequest = components["schemas"]["StartSessionRequest"];

/** Response of GET /api/v1/pomodoro/current. */
export type CurrentSessionResponse =
  paths["/api/v1/pomodoro/current"]["get"]["responses"]["200"]["content"]["application/json"];

/** Starts a new pomodoro session. Cancels any existing session. */
export async function startSession(req: StartSessionRequest): Promise<PomodoroSession> {
  return request<PomodoroSession>("/pomodoro/start", {
    method: "POST",
    body: JSON.stringify(req),
    action: "start session",
  });
}

/** Gets the current session state. */
export async function getCurrentSession(): Promise<CurrentSessionResponse> {
  return request<CurrentSessionResponse>("/pomodoro/current", {
    action: "get current session",
  });
}

/** Pauses the current session. */
export async function pauseSession(): Promise<PomodoroSession> {
  return request<PomodoroSession>("/pomodoro/pause", {
    method: "POST",
    action: "pause session",
  });
}

/** Resumes the current session. */
export async function resumeSession(): Promise<PomodoroSession> {
  return request<PomodoroSession>("/pomodoro/resume", {
    method: "POST",
    action: "resume session",
  });
}

/** Stops (completes/cancels) the current session. */
export async function stopSession(): Promise<PomodoroSession> {
  return request<PomodoroSession>("/pomodoro/stop", {
    method: "POST",
    action: "stop session",
  });
}
