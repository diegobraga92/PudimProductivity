/** Splits a "#rrggbb" hex color into its RGB channels. */
function hexToRgb(hex: string): { r: number; g: number; b: number } {
  return {
    r: parseInt(hex.slice(1, 3), 16),
    g: parseInt(hex.slice(3, 5), 16),
    b: parseInt(hex.slice(5, 7), 16),
  };
}

/** "rgba(r, g, b, a)" from a "#rrggbb" hex color and an alpha in [0, 1]. */
export function hexToRgba(hex: string, alpha: number): string {
  const { r, g, b } = hexToRgb(hex);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

/** True when a hex color is light enough to need dark text on top. */
export function isLightColor(hex: string): boolean {
  const { r, g, b } = hexToRgb(hex);
  return 0.299 * r + 0.587 * g + 0.114 * b > 160;
}
