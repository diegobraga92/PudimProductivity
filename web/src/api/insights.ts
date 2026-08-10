import config from "../config";
import type { components } from "./generated/insights-v1";

// Phase 9a: AI coach — weekly productivity report (api/openapi/insights-v1.yaml).

export type InsightReport = components["schemas"]["InsightReport"];

/**
 * Fetches the weekly productivity report. `date` is optional (any date in the
 * target week; defaults to today).
 */
export async function getWeeklyInsights(date?: string): Promise<InsightReport> {
  const query = date ? `?date=${encodeURIComponent(date)}` : "";
  const response = await fetch(`${config.apiBaseUrl}/insights/weekly${query}`);

  if (!response.ok) {
    throw new Error(`Failed to fetch weekly insights: ${response.status}`);
  }
  return response.json() as Promise<InsightReport>;
}
