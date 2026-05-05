import config from "../config";

export interface HealthResponse {
  status: "ok" | "degraded" | "down";
  version: string;
  db: "connected" | "disconnected";
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
