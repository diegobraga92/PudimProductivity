import config from "../config";
import type { components, paths } from "./generated/pomodoro-v1";

// Types are generated from api/openapi/pomodoro-v1.yaml (the source of truth).
export type SessionStatus = components["schemas"]["SessionStatus"];
export type PomodoroSession = components["schemas"]["PomodoroSession"];
export type StartSessionRequest = components["schemas"]["StartSessionRequest"];

/** Response of GET /api/v1/pomodoro/current — an active-session discriminated union. */
export type CurrentSessionResponse =
  paths["/api/v1/pomodoro/current"]["get"]["responses"]["200"]["content"]["application/json"];

/**
 * Starts a new pomodoro session. Cancels any existing session.
 */
export async function startSession(req: StartSessionRequest): Promise<PomodoroSession> {
  const response = await fetch(`${config.apiBaseUrl}/pomodoro/start`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });

  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to start session: ${response.status}`);
  }

  return response.json() as Promise<PomodoroSession>;
}

/**
 * Gets the current session state.
 */
export async function getCurrentSession(): Promise<CurrentSessionResponse> {
  const response = await fetch(`${config.apiBaseUrl}/pomodoro/current`);

  if (!response.ok) {
    throw new Error(`Failed to get current session: ${response.status}`);
  }

  return response.json() as Promise<CurrentSessionResponse>;
}

/**
 * Pauses the current session.
 */
export async function pauseSession(): Promise<PomodoroSession> {
  const response = await fetch(`${config.apiBaseUrl}/pomodoro/pause`, {
    method: "POST",
  });

  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to pause session: ${response.status}`);
  }

  return response.json() as Promise<PomodoroSession>;
}

/**
 * Resumes the current session.
 */
export async function resumeSession(): Promise<PomodoroSession> {
  const response = await fetch(`${config.apiBaseUrl}/pomodoro/resume`, {
    method: "POST",
  });

  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to resume session: ${response.status}`);
  }

  return response.json() as Promise<PomodoroSession>;
}

/**
 * Stops (completes/cancels) the current session.
 */
export async function stopSession(): Promise<PomodoroSession> {
  const response = await fetch(`${config.apiBaseUrl}/pomodoro/stop`, {
    method: "POST",
  });

  if (!response.ok) {
    const err = await response.json().catch(() => ({ error: "Unknown error" }));
    throw new Error(err.error || `Failed to stop session: ${response.status}`);
  }

  return response.json() as Promise<PomodoroSession>;
}
