import { request } from "./client";
import type { components } from "./generated/library-v1";

// Types are generated from api/openapi/library-v1.yaml.
export type LibraryItem = components["schemas"]["LibraryItem"];
export type MediaType = components["schemas"]["MediaType"];
export type CreateLibraryItemRequest = components["schemas"]["CreateLibraryItemRequest"];
export type UpdateLibraryItemRequest = components["schemas"]["UpdateLibraryItemRequest"];
export type ImportResult = components["schemas"]["ImportResult"];
export type ScoreCandidate = components["schemas"]["ScoreCandidate"];

export async function listLibraryItems(
  type?: MediaType,
  done?: boolean,
  subtype?: string,
): Promise<LibraryItem[]> {
  const params = new URLSearchParams();
  if (type) params.set("type", type);
  if (done !== undefined) params.set("done", String(done));
  if (subtype) params.set("subtype", subtype);
  const qs = params.toString();
  return request<LibraryItem[]>(`/library${qs ? `?${qs}` : ""}`, { action: "list library" });
}

/** Distinct genre/console values for the subtype filter, optionally by type. */
export async function listLibrarySubtypes(type?: MediaType): Promise<string[]> {
  const qs = type ? `?type=${encodeURIComponent(type)}` : "";
  return request<string[]>(`/library/subtypes${qs}`, { action: "list subtypes" });
}

export async function createLibraryItem(req: CreateLibraryItemRequest): Promise<LibraryItem> {
  return request<LibraryItem>("/library", {
    method: "POST",
    body: JSON.stringify(req),
    action: "add item",
  });
}

export async function updateLibraryItem(
  itemId: string,
  req: UpdateLibraryItemRequest,
): Promise<LibraryItem> {
  return request<LibraryItem>(`/library/${itemId}`, {
    method: "PUT",
    body: JSON.stringify(req),
    action: "update item",
  });
}

export async function deleteLibraryItem(itemId: string): Promise<void> {
  return request<void>(`/library/${itemId}`, { method: "DELETE", action: "delete item" });
}

export async function importLibraryItems(req: {
  items: CreateLibraryItemRequest[];
}): Promise<ImportResult> {
  return request<ImportResult>("/library/import", {
    method: "POST",
    body: JSON.stringify(req),
    action: "import items",
  });
}

/** Search the configured rating provider (OMDb/RAWG) for a title; the user confirms the match. */
export async function searchLibraryScores(
  query: string,
  type: MediaType,
  year?: number,
): Promise<ScoreCandidate[]> {
  const params = new URLSearchParams({ query, type });
  if (year != null) params.set("year", String(year));
  return request<ScoreCandidate[]>(`/library/score/search?${params}`, {
    action: "look up score",
  });
}

export type ScoreBatchItem = components["schemas"]["ScoreBatchItem"];
export type ScoreBatchResponse = components["schemas"]["ScoreBatchResponse"];

/** Batch score lookup (used by CSV auto-scoring); per-item failures come back inline. */
export async function searchLibraryScoresBatch(
  items: ScoreBatchItem[],
): Promise<ScoreBatchResponse> {
  return request<ScoreBatchResponse>("/library/score/batch", {
    method: "POST",
    body: JSON.stringify({ items }),
    action: "look up scores",
  });
}
