/**
 * Copies the built web app (../web/dist) into desktop/dist-web/ so electron-builder
 * can package it inside the asar next to dist-electron/ (the packaged main
 * process resolves it via `path.join(__dirname, "..", "dist-web")`).
 */
import { cpSync, existsSync, mkdirSync, rmSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const src = resolve(here, "..", "..", "web", "dist");
const dest = resolve(here, "..", "dist-web");

if (!existsSync(src)) {
  console.error(`[desktop] web build not found at ${src}. Run: npm --prefix ../web run build:desktop`);
  process.exit(1);
}

rmSync(dest, { recursive: true, force: true });
mkdirSync(dest, { recursive: true });
cpSync(src, dest, { recursive: true });
console.log(`[desktop] copied ${src} -> ${dest}`);
