import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import { visualizer } from "rollup-plugin-visualizer";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

/**
 * Reads the canonical app version from the `VERSION` file one level above the
 * web project (see scripts/sync-version.mjs).
 */
function readAppVersion(): string {
  const file = join(dirname(fileURLToPath(import.meta.url)), "..", "VERSION");
  try {
    return readFileSync(file, "utf8").trim();
  } catch {
    throw new Error(
      `Cannot read the app version from ${file}: the repository-root VERSION file ` +
        "must be available next to the web/ directory (see web/Dockerfile).",
    );
  }
}

const appVersion = readAppVersion();

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");

  const backendPort = env.BACKEND_PORT ?? "8080";
  const frontendPort = Number(env.FRONTEND_PORT) || 3000;
  const analyze = env.ANALYZE === "1";

  return {
    define: { __APP_VERSION__: JSON.stringify(appVersion) },
    // rollup-plugin-visualizer is opt-in: `ANALYZE=1 npm run build` emits
    // dist/stats.html with a treemap of chunk sizes (see package.json
    // build:analyze). Normal builds are unaffected.
    plugins: [react(), ...(analyze ? [visualizer({ filename: "dist/stats.html", gzipSize: true, open: false })] : [])],
    server: {
      port: frontendPort,
      // The shared i18n dictionary lives one level above the web project.
      fs: { allow: [".."] },
      proxy: {
        "/api": {
          target: `http://localhost:${backendPort}`,
          changeOrigin: true,
          // Enable WebSocket proxying for the real-time sync endpoint.
          ws: true,
        },
      },
    },
    build: {
      outDir: "dist",
      sourcemap: true,
    },
  };
});