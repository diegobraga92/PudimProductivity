/**
 * Generates the app/tray icons as PNGs without any image tooling, so the
 * desktop package works out of the box. Replace with real artwork later:
 *   - assets/icon.png        512x512 app/window icon
 *   - assets/trayTemplate.png 32x32  tray icon (black + alpha, macOS template)
 */
import { deflateSync } from "node:zlib";
import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const assetsDir = join(dirname(fileURLToPath(import.meta.url)), "..", "assets");
mkdirSync(assetsDir, { recursive: true });

// ── Minimal PNG encoder (RGBA, 8-bit, no interlace) ─────────────────────────
const CRC_TABLE = new Uint32Array(256);
for (let n = 0; n < 256; n++) {
  let c = n;
  for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  CRC_TABLE[n] = c >>> 0;
}

function crc32(buf) {
  let crc = 0xffffffff;
  for (let i = 0; i < buf.length; i++) crc = CRC_TABLE[(crc ^ buf[i]) & 0xff] ^ (crc >>> 8);
  return (crc ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length, 0);
  const body = Buffer.concat([Buffer.from(type, "ascii"), data]);
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(body), 0);
  return Buffer.concat([len, body, crc]);
}

function encodePng(width, height, rgba) {
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0);
  ihdr.writeUInt32BE(height, 4);
  ihdr[8] = 8; // bit depth
  ihdr[9] = 6; // color type: RGBA
  const stride = width * 4 + 1; // +1 filter byte per row
  const raw = Buffer.alloc(stride * height);
  for (let y = 0; y < height; y++) {
    raw[y * stride] = 0; // filter: None
    rgba.copy(raw, y * stride + 1, y * width * 4, (y + 1) * width * 4);
  }
  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]),
    chunk("IHDR", ihdr),
    chunk("IDAT", deflateSync(raw)),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

/** Filled circle (with a soft rim) centered on a transparent canvas. */
function drawIcon(size, rgb, rimRgb, radiusFraction) {
  const px = Buffer.alloc(size * size * 4);
  const c = size / 2;
  const r = size * radiusFraction;
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const dx = x + 0.5 - c;
      const dy = y + 0.5 - c;
      const d = Math.sqrt(dx * dx + dy * dy);
      const i = (y * size + x) * 4;
      if (d <= r) {
        const color = d > r * 0.88 ? rimRgb : rgb;
        px[i] = color[0];
        px[i + 1] = color[1];
        px[i + 2] = color[2];
        px[i + 3] = 255;
      } else {
        px[i + 3] = 0; // transparent
      }
    }
  }
  return encodePng(size, size, px);
}

// Pudim-ish peach/red brand color on a darker rim.
writeFileSync(
  join(assetsDir, "icon.png"),
  drawIcon(512, [0xe5, 0x8b, 0x76], [0x8a, 0x2a, 0x2a], 0.42),
);
// Tray icon: solid black + alpha (macOS template convention; fine everywhere).
writeFileSync(
  join(assetsDir, "trayTemplate.png"),
  drawIcon(32, [0x00, 0x00, 0x00], [0x00, 0x00, 0x00], 0.45),
);

console.log(`[desktop] generated assets in ${assetsDir}`);
