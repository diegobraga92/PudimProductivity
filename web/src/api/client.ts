import config from "../config";

export interface HealthResponse {
  status: "ok" | "degraded" | "down";
  version: string;
  db: "connected" | "disconnected";
}

/**
 * Development-only auth headers, mirroring the backend's `shared.AuthMiddleware`
 * dev-mode contract (X-User-ID / X-User-Role). The backend currently trusts
 * these headers; in production they would be replaced by a JWT/session credential.
 *
 * Mutating endpoints (POST/PUT/DELETE) are protected by `RequireRole("admin", "user")`,
 * so clients must present these headers to create/update/delete tasks.
 */
export const DEV_USER_ID = "dev-user";
export const DEV_USER_ROLE = "user";

/** Returns headers for API calls, defaulting to the dev user identity. */
export function apiHeaders(extra?: Record<string, string>): Record<string, string> {
  return {
    "Content-Type": "application/json",
    "X-User-ID": DEV_USER_ID,
    "X-User-Role": DEV_USER_ROLE,
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
