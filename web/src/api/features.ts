import { request } from "./client";

export interface FeatureFlag {
  id: string;
  name: string;
  description?: string;
  enabled: boolean;
}

export async function getFeature(name: string): Promise<FeatureFlag> {
  return request<FeatureFlag>(`/features/${encodeURIComponent(name)}`, {
    action: "get feature flag",
  });
}
