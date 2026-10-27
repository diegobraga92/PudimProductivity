/**
 * Helpers for manual (drag and drop) ordering. Kept pure so the browser code
 * stays a thin layer over functions that are easy to test.
 */

/** Returns a copy of items with the entry at `from` moved to index `to`. */
export function moveItem<T>(items: readonly T[], from: number, to: number): T[] {
  if (from < 0 || from >= items.length) return [...items];
  const next = [...items];
  const [moved] = next.splice(from, 1);
  next.splice(clamp(to, 0, next.length), 0, moved);
  return next;
}

/**
 * Index the dragged item lands on, given the vertical midpoints of the other
 * items in visual order and the pointer's Y. The item sits after every row
 * whose midpoint is above the pointer.
 */
export function landingIndex(midpoints: readonly number[], pointerY: number): number {
  let index = 0;
  for (const midpoint of midpoints) {
    if (pointerY > midpoint) index += 1;
  }
  return index;
}

/** Maps each id to its 0-based rank, matching the server's `position`. */
export function positionsFor(ids: readonly string[]): Map<string, number> {
  return new Map(ids.map((id, index) => [id, index]));
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}
