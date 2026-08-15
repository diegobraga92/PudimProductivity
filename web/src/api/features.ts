import config from "../config";
import { apiHeaders } from "./client";

export interface FeatureFlag {
  id: string;
  name: string;
  description?: string;
  enabled: boolean;
}

export async function listEnabledFeatures(): Promise<FeatureFlag[]> {
  const response = await fetch(`${config.apiBaseUrl}/features`);

  if (!response.ok) {
    throw new Error(`Failed to list feature flags: ${response.status}`);
  }

  return response.json() as Promise<FeatureFlag[]>;
}

export async function getFeature(name: string): Promise<FeatureFlag> {
  const response = await fetch(`${config.apiBaseUrl}/features/${encodeURIComponent(name)}`);

  if (!response.ok) {
    throw new Error(`Failed to get feature flag: ${response.status}`);
  }

  return response.json() as Promise<FeatureFlag>;
}

/**
 * Toggles a feature flag. Admin-only on the backend (RequireRole("admin")); the
 * web client sends the dev role from localStorage, so this requires the user to
 * have switched to admin mode (see ServerSettings page).
 */
export async function toggleFeature(name: string, enabled: boolean): Promise<FeatureFlag> {
  const response = await fetch(`${config.apiBaseUrl}/features/${encodeURIComponent(name)}/toggle`, {
    method: "PUT",
    headers: apiHeaders(),
    body: JSON.stringify({ enabled }),
  });

  if (!response.ok) {
    throw new Error(`Failed to toggle feature flag: ${response.status}`);
  }

  return response.json() as Promise<FeatureFlag>;
}