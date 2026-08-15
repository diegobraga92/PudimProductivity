import config from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/library-v1";

// Types are generated from api/openapi/library-v1.yaml.
export type LibraryItem = components["schemas"]["LibraryItem"];
export type MediaType = components["schemas"]["MediaType"];
export type CreateLibraryItemRequest = components["schemas"]["CreateLibraryItemRequest"];
export type UpdateLibraryItemRequest = components["schemas"]["UpdateLibraryItemRequest"];
export type ImportResult = components["schemas"]["ImportResult"];
export type ScoreCandidate = components["schemas"]["ScoreCandidate"];

async function handleError(response: Response, fallback: string): Promise<never> {
  const body = await response.json().catch(() => null);
  throw new Error(body?.error || fallback);
}

export async function listLibraryItems(type?: MediaType, done?: boolean, subtype?: string): Promise<LibraryItem[]> {
  const params = new URLSearchParams();
  if (type) params.set("type", type);
  if (done !== undefined) params.set("done", String(done));
  if (subtype) params.set("subtype", subtype);
  const qs = params.toString();
  const res = await fetch(`${config.apiBaseUrl}/library${qs ? `?${qs}` : ""}`);
  if (!res.ok) await handleError(res, `Failed to list library: ${res.status}`);
  return res.json() as Promise<LibraryItem[]>;
}

/** Distinct genre/console values for the subtype filter, optionally by type. */
export async function listLibrarySubtypes(type?: MediaType): Promise<string[]> {
  const qs = type ? `?type=${encodeURIComponent(type)}` : "";
  const res = await fetch(`${config.apiBaseUrl}/library/subtypes${qs}`);
  if (!res.ok) await handleError(res, `Failed to list subtypes: ${res.status}`);
  return res.json() as Promise<string[]>;
}

export async function createLibraryItem(req: CreateLibraryItemRequest): Promise<LibraryItem> {
  const res = await fetch(`${config.apiBaseUrl}/library`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to add item: ${res.status}`);
  return res.json() as Promise<LibraryItem>;
}

export async function updateLibraryItem(
  itemId: string,
  req: UpdateLibraryItemRequest,
): Promise<LibraryItem> {
  const res = await fetch(`${config.apiBaseUrl}/library/${itemId}`, {
    method: "PUT",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to update item: ${res.status}`);
  return res.json() as Promise<LibraryItem>;
}

export async function deleteLibraryItem(itemId: string): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/library/${itemId}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to delete item: ${res.status}`);
}

export async function importLibraryItems(req: {
  items: CreateLibraryItemRequest[];
}): Promise<ImportResult> {
  const res = await fetch(`${config.apiBaseUrl}/library/import`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to import items: ${res.status}`);
  return res.json() as Promise<ImportResult>;
}

/**
 * Searches the configured rating provider (OMDb for films/series, RAWG for
 * games) for a title. The returned candidates let the user confirm the right
 * match before its score is saved with the item.
 */
export async function searchLibraryScores(
  query: string,
  type: MediaType,
  year?: number,
): Promise<ScoreCandidate[]> {
  const params = new URLSearchParams({ query, type });
  if (year != null) params.set("year", String(year));
  const res = await fetch(`${config.apiBaseUrl}/library/score/search?${params}`);
  if (!res.ok) await handleError(res, `Failed to look up score: ${res.status}`);
  return res.json() as Promise<ScoreCandidate[]>;
}

export type ScoreBatchItem = components["schemas"]["ScoreBatchItem"];
export type ScoreBatchResponse = components["schemas"]["ScoreBatchResponse"];

/**
 * Looks up scores for many titles at once (used by the CSV import auto-scoring
 * flow). Per-item failures come back inline in each result's error field.
 */
export async function searchLibraryScoresBatch(items: ScoreBatchItem[]): Promise<ScoreBatchResponse> {
  const res = await fetch(`${config.apiBaseUrl}/library/score/batch`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify({ items }),
  });
  if (!res.ok) await handleError(res, `Failed to look up scores: ${res.status}`);
  return res.json() as Promise<ScoreBatchResponse>;
}
