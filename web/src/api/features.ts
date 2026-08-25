import config from "../config";

export interface FeatureFlag {
  id: string;
  name: string;
  description?: string;
  enabled: boolean;
}

export async function getFeature(name: string): Promise<FeatureFlag> {
  const response = await fetch(`${config.apiBaseUrl}/features/${encodeURIComponent(name)}`);

  if (!response.ok) {
    throw new Error(`Failed to get feature flag: ${response.status}`);
  }

  return response.json() as Promise<FeatureFlag>;
}