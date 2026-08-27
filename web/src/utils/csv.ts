/**
 * Minimal CSV parser (RFC 4180-ish): handles quoted fields, escaped quotes
 * (""), commas and newlines inside quotes, CRLF/LF line endings and an
 * optional UTF-8 BOM. Returns rows as arrays of strings. Trailing empty rows
 * (e.g. from a final newline) are dropped.
 */
export function parseCsv(text: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let field = "";
  let inQuotes = false;

  const src = text.charCodeAt(0) === 0xfeff ? text.slice(1) : text;

  for (let i = 0; i < src.length; i++) {
    const c = src[i];
    if (inQuotes) {
      if (c === '"') {
        if (src[i + 1] === '"') {
          field += '"';
          i++;
        } else {
          inQuotes = false;
        }
      } else {
        field += c;
      }
    } else if (c === '"') {
      inQuotes = true;
    } else if (c === ",") {
      row.push(field);
      field = "";
    } else if (c === "\n" || c === "\r") {
      if (c === "\r" && src[i + 1] === "\n") i++;
      row.push(field);
      field = "";
      rows.push(row);
      row = [];
    } else {
      field += c;
    }
  }

  if (field !== "" || row.length > 0) {
    row.push(field);
    rows.push(row);
  }

  while (rows.length > 0 && rows[rows.length - 1].every((f) => f.trim() === "")) {
    rows.pop();
  }
  return rows;
}

/** Normalizes a CSV media-type cell to a valid Library media type, or null. */
export type MediaTypeValue = "movie" | "series" | "book" | "game";

const MEDIA_TYPE_VALUES: readonly MediaTypeValue[] = ["movie", "series", "book", "game"];

export function normalizeMediaType(value: string): MediaTypeValue | null {
  const v = value.trim().toLowerCase() as MediaTypeValue;
  return MEDIA_TYPE_VALUES.includes(v) ? v : null;
}

/** Parses a CSV "done" cell into a boolean using common truthy tokens. */
export function parseDoneValue(value: string): boolean {
  const v = value.trim().toLowerCase();
  return ["true", "yes", "y", "1", "x", "done", "read", "watched", "played", "✔"].includes(v);
}

/** Parses a CSV release-year cell into an integer, or null when empty/garbage. */
export function parseYearValue(value: string): number | null {
  const v = value.trim();
  if (v === "") return null;
  const n = Number(v);
  if (!Number.isInteger(n) || n < 1800 || n > 2100) return null;
  return n;
}

/** Parses a CSV score cell into a number on the 0-100 scale, or null. */
export function parseScoreValue(value: string): number | null {
  const v = value.trim();
  if (v === "") return null;
  const n = Number(v);
  if (!Number.isFinite(n) || n < 0 || n > 100) return null;
  return n;
}

// --- Auto-score batch application (CSV import) ---

export interface AutoScoredValue {
  score: number;
  score_source: string;
  release_year?: number;
}

export interface BatchScoreResult {
  index: number;
  candidates?: { score: number; score_source: string; year?: number }[] | null;
  error?: string | null;
}

/**
 * Applies one batch score-lookup response to the requested rows. `targets`
 * maps each batch request index to a data-row index (the batch endpoint echoes
 * the request position). The top candidate fills the row (score, source and,
 * when the provider reports one, the release year — matching the single-score
 * flow); rows with no match (no error, empty candidates) are counted
 * separately from failed lookups.
 */
export function applyAutoScoreResults(
  targets: { requestIndex: number; row: number }[],
  results: BatchScoreResult[],
): { fill: Record<number, AutoScoredValue>; found: number; noRating: number } {
  const byIndex = new Map(targets.map((t) => [t.requestIndex, t.row]));
  const fill: Record<number, AutoScoredValue> = {};
  let found = 0;
  let noRating = 0;
  for (const res of results) {
    const row = byIndex.get(res.index);
    if (row === undefined) continue;
    if (res.error || !res.candidates || res.candidates.length === 0) {
      if (!res.error) noRating++;
      continue;
    }
    const top = res.candidates[0];
    fill[row] = { score: top.score, score_source: top.score_source };
    if (top.year != null && top.year > 0) {
      fill[row].release_year = top.year;
    }
    found++;
  }
  return { fill, found, noRating };
}
