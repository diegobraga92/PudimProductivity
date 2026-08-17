/**
 * Generates the app/tray icons from their SVG sources using @resvg/resvg-js
 * (a small, prebuilt SVG -> PNG rasterizer; build-time devDependency only).
 *   - assets/icon.svg  -> assets/icon.png         512x512 app/window icon
 *   - assets/tray.svg  -> assets/trayTemplate.png  32x32  tray icon
 */
import { Resvg } from "@resvg/resvg-js";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const assetsDir = join(dirname(fileURLToPath(import.meta.url)), "..", "assets");
mkdirSync(assetsDir, { recursive: true });

/** Rasterizes `svgName` at `size`x`size` and writes `pngName` into assets/. */
function render(svgName, pngName, size) {
  const svg = readFileSync(join(assetsDir, svgName), "utf8");
  const resvg = new Resvg(svg, {
    fitTo: { mode: "width", value: size },
  });
  const png = resvg.render().asPng();
  writeFileSync(join(assetsDir, pngName), png);
  console.log(`[desktop] generated ${pngName} (${size}x${size})`);
}

render("icon.svg", "icon.png", 512);
render("tray.svg", "trayTemplate.png", 32);

