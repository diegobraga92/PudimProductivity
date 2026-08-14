/// <reference types="vite/client" />

const config = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? "/api/v1",
  // Public base URL for uploaded media objects (e.g. the S3 bucket or CDN
  // domain). Recipe image keys returned by the upload flow are resolved
  // against this. Leave empty when no media backend is configured.
  mediaBaseUrl: import.meta.env.VITE_MEDIA_BASE_URL ?? "",
} as const;

export default config;
