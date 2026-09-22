import config from "../config";
import type { components } from "./generated/health-v1";

export type HealthResponse = components["schemas"]["HealthResponse"];

/**
 * Dev-mode identity headers mirroring the backend's AuthMiddleware
 * (X-User-ID / X-User-Role).
 */
export const DEV_USER_ID = "dev-user";
const DEV_USER_ROLE = "user";

const DEV_ROLE_KEY = "devRole";

/** Returns the active dev role ("user" | "admin"), defaulting to "user". */
export function getDevRole(): string {
  const role = localStorage.getItem(DEV_ROLE_KEY);
  return role === "admin" || role === "user" ? role : DEV_USER_ROLE;
}

/** Persists the dev identity role used in the X-User-Role header. */
export function setDevRole(role: "admin" | "user") {
  localStorage.setItem(DEV_ROLE_KEY, role);
}

/**
 * Returns identity headers without a Content-Type, for requests whose body sets
 * its own. Most importantly FormData uploads, where the browser has to add the
 * multipart boundary itself.
 */
export function apiIdentityHeaders(extra?: Record<string, string>): Record<string, string> {
  return {
    "X-User-ID": DEV_USER_ID,
    "X-User-Role": getDevRole(),
    ...extra,
  };
}

/** Returns headers for JSON API calls, defaulting to the dev user identity. */
export function apiHeaders(extra?: Record<string, string>): Record<string, string> {
  return { "Content-Type": "application/json", ...apiIdentityHeaders(), ...extra };
}

/**
 * Fetches the health status from the backend.
 */
export async function getHealth(): Promise<HealthResponse> {
  const response = await fetch(`${config.apiBaseUrl}/health`);

  if (!response.ok && response.status !== 503) {
    throw new Error(`Health check failed: ${response.status}`);
  }

  return response.json() as Promise<HealthResponse>;
}
