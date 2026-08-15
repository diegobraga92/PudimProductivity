import config from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/admin-v1";

// Types are generated from api/openapi/admin-v1.yaml.
export type ScoreProvidersConfig = components["schemas"]["ScoreProvidersConfig"];
export type ScoreProvidersUpdate = components["schemas"]["ScoreProvidersUpdate"];

async function handleError(response: Response, fallback: string): Promise<never> {
  const body = await response.json().catch(() => null);
  throw new Error(body?.error || fallback);
}

/** Reads the effective score-provider configuration (API keys masked). */
export async function getScoreProviders(): Promise<ScoreProvidersConfig> {
  const res = await fetch(`${config.apiBaseUrl}/admin/score-providers`, {
    // Admin GETs must still send the dev identity headers: the backend defaults
    // requests without X-User-Role to "user", which would 403 here.
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to load score provider settings: ${res.status}`);
  return res.json() as Promise<ScoreProvidersConfig>;
}

/** Validates, applies and persists a new score-provider configuration. */
export async function saveScoreProviders(req: ScoreProvidersUpdate): Promise<ScoreProvidersConfig> {
  const res = await fetch(`${config.apiBaseUrl}/admin/score-providers`, {
    method: "PUT",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to save score provider settings: ${res.status}`);
  return res.json() as Promise<ScoreProvidersConfig>;
}
