import config from "../config";
import type { components } from "./generated/health-v1";

export type HealthResponse = components["schemas"]["HealthResponse"];

/**
 * Development-only auth headers, mirroring the backend's `shared.AuthMiddleware`
 * dev-mode contract (X-User-ID / X-User-Role). The backend currently trusts
 * these headers; in production they would be replaced by a JWT/session credential.
 *
 * Mutating endpoints (POST/PUT/DELETE) are protected by `RequireRole("admin", "user")`,
 * so clients must present these headers to create/update/delete tasks. Admin-only
 * endpoints (feature-flag toggles, score-provider settings) additionally require
 * role "admin" — the dev role is stored in localStorage so the admin UI can switch
 * it (see `setDevRole`).
 */
export const DEV_USER_ID = "dev-user";
export const DEV_USER_ROLE = "user";

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

/** Returns headers for API calls, defaulting to the dev user identity. */
export function apiHeaders(extra?: Record<string, string>): Record<string, string> {
  return {
    "Content-Type": "application/json",
    "X-User-ID": DEV_USER_ID,
    "X-User-Role": getDevRole(),
    ...extra,
  };
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
