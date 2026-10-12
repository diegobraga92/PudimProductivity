import config from "../config";
import type { components } from "./generated/health-v1";

export type HealthResponse = components["schemas"]["HealthResponse"];

/** Dev identity headers mirroring the backend AuthMiddleware (X-User-ID / X-User-Role). */
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

/** Identity headers without Content-Type, for bodies that set their own (e.g. FormData uploads). */
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

/** Backend health status. */
export async function getHealth(): Promise<HealthResponse> {
  const response = await fetch(`${config.apiBaseUrl}/health`);

  if (!response.ok && response.status !== 503) {
    throw new Error(`Health check failed: ${response.status}`);
  }

  return response.json() as Promise<HealthResponse>;
}

/** Turns an error response body (`{ error }` from the backend) into an Error. */
export async function apiError(response: Response, fallback: string): Promise<Error> {
  const body = await response.json().catch(() => null);
  return new Error(body?.error || fallback);
}

export interface RequestOptions {
  /** Short verb phrase used in the generic error, e.g. "list recipes". */
  action: string;
  method?: string;
  /** Overrides the default JSON identity headers (e.g. multipart uploads). */
  headers?: Record<string, string>;
  body?: BodyInit | null;
  /** Message thrown for 404 responses (defaults to the generic message). */
  notFound?: string;
  /** Message thrown for 413 responses (reverse proxies reply with an HTML page). */
  tooLarge?: string;
}

/** Single entry point for JSON API calls. */
export async function request<T>(path: string, opts: RequestOptions): Promise<T> {
  const { action, notFound, tooLarge, headers, ...init } = opts;
  const response = await fetch(`${config.apiBaseUrl}${path}`, {
    ...init,
    headers: headers ?? apiHeaders(),
  });
  if (!response.ok) {
    if (notFound && response.status === 404) throw new Error(notFound);
    if (tooLarge && response.status === 413) throw new Error(tooLarge);
    throw await apiError(response, `Failed to ${action}: ${response.status}`);
  }
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}
