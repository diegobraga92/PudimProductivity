/// <reference types="vite/client" />

// Backend URL injected by the desktop shell (additionalArguments -> contextBridge) when present.
const desktopApiBaseUrl = window.desktop?.getApiBaseUrl?.();

const config = {
  apiBaseUrl: desktopApiBaseUrl || import.meta.env.VITE_API_BASE_URL || "/api/v1",
  mediaBaseUrl: import.meta.env.VITE_MEDIA_BASE_URL ?? "",
} as const;

export default config;

// Absolute API origin, for relative bases like "/api/v1" too.
export function apiOrigin(): string {
  return new URL(config.apiBaseUrl, window.location.origin).origin;
}

// Base URL for stored media keys: the S3/CDN base when configured, else derived from the API base.
export function mediaBaseUrl(): string {
  if (config.mediaBaseUrl) return config.mediaBaseUrl.replace(/\/+$/, "");
  const base = config.apiBaseUrl.replace(/\/+$/, "");
  return new URL(`${base}/media`, window.location.origin).toString().replace(/\/+$/, "");
}
