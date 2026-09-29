/**
 * Guards the compiled preload against a runtime-only failure.
 *
 * desktop/src/preload.ts runs with webPreferences.sandbox = true, where Electron's
 * `require` only resolves a whitelist of built-ins. A relative require (such as the
 * `require("./generated-version")` that shipped in 1.1.0) throws "module not found",
 * the preload is discarded, window.desktop stays undefined and every desktop-only
 * feature disappears with no visible error. tsc cannot catch that, so verify the
 * emitted file right after `tsc`.
 */
import { readFileSync } from "node:fs";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const preload = resolve(here, "..", "dist-electron", "preload.js");
const label = relative(process.cwd(), preload);

const source = readFileSync(preload, "utf8");
const offenders = [...source.matchAll(/require\(\s*["']\.\.?\/[^"']*["']\s*\)/g)].map((m) => m[0]);

if (offenders.length > 0) {
  console.error(`[desktop] ${label} must be self-contained, found: ${offenders.join(", ")}`);
  console.error(
    "[desktop] Sandboxed preloads cannot load other files. Inline the value or pass it\n" +
      "[desktop] through additionalArguments in desktop/src/main.ts."
  );
  process.exit(1);
}

console.log(`[desktop] preload is self-contained (${label})`);
