import config from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/library-v1";

// Types are generated from api/openapi/library-v1.yaml.
export type LibraryItem = components["schemas"]["LibraryItem"];
export type MediaType = components["schemas"]["MediaType"];
export type CreateLibraryItemRequest = components["schemas"]["CreateLibraryItemRequest"];
export type UpdateLibraryItemRequest = components["schemas"]["UpdateLibraryItemRequest"];
export type ImportResult = components["schemas"]["ImportResult"];

async function handleError(response: Response, fallback: string): Promise<never> {
  const body = await response.json().catch(() => null);
  throw new Error(body?.error || fallback);
}

export async function listLibraryItems(type?: MediaType, done?: boolean): Promise<LibraryItem[]> {
  const params = new URLSearchParams();
  if (type) params.set("type", type);
  if (done !== undefined) params.set("done", String(done));
  const qs = params.toString();
  const res = await fetch(`${config.apiBaseUrl}/library${qs ? `?${qs}` : ""}`);
  if (!res.ok) await handleError(res, `Failed to list library: ${res.status}`);
  return res.json() as Promise<LibraryItem[]>;
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
