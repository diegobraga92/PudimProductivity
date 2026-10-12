import { request } from "./client";
import type { components } from "./generated/admin-v1";

// Types are generated from api/openapi/admin-v1.yaml.
export type ScoreProvidersConfig = components["schemas"]["ScoreProvidersConfig"];
export type ScoreProvidersUpdate = components["schemas"]["ScoreProvidersUpdate"];

/** Reads the effective score-provider configuration (API keys masked). */
export async function getScoreProviders(): Promise<ScoreProvidersConfig> {
  // Admin GETs need the identity headers: without X-User-Role the backend assumes "user" and 403s.
  return request<ScoreProvidersConfig>("/admin/score-providers", {
    action: "load score provider settings",
  });
}

/** Validates, applies and persists a new score-provider configuration. */
export async function saveScoreProviders(
  req: ScoreProvidersUpdate,
): Promise<ScoreProvidersConfig> {
  return request<ScoreProvidersConfig>("/admin/score-providers", {
    method: "PUT",
    body: JSON.stringify(req),
    action: "save score provider settings",
  });
}
