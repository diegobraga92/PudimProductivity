/// <reference types="vite/client" />

// Electron desktop builds receive the backend URL from the shell (passed via
// additionalArguments -> contextBridge).
const desktopApiBaseUrl = window.desktop?.getApiBaseUrl?.();

const config = {
  apiBaseUrl: desktopApiBaseUrl || import.meta.env.VITE_API_BASE_URL || "/api/v1",
  // Public base URL for uploaded media objects.
  mediaBaseUrl: import.meta.env.VITE_MEDIA_BASE_URL ?? "",
} as const;

export default config;

// Absolute origin of the API, for relative bases like "/api/v1" and for the
// absolute base the desktop shell passes.
export function apiOrigin(): string {
  return new URL(config.apiBaseUrl, window.location.origin).origin;
}

// Base URL for stored media keys. Uses the configured S3/CDN base when set,
// otherwise derives it from the API base served by local storage.
export function mediaBaseUrl(): string {
  if (config.mediaBaseUrl) return config.mediaBaseUrl.replace(/\/+$/, "");
  const base = config.apiBaseUrl.replace(/\/+$/, "");
  return new URL(`${base}/media`, window.location.origin).toString().replace(/\/+$/, "");
}
