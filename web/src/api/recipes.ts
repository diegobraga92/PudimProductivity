import type { QueryClient } from "@tanstack/react-query";
import config, { apiOrigin, mediaBaseUrl } from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/recipes-v1";

// Types are generated from api/openapi/recipes-v1.yaml (the source of truth).
export type Recipe = components["schemas"]["Recipe"];
export type CreateRecipeRequest = components["schemas"]["CreateRecipeRequest"];
export type UploadURLRequest = components["schemas"]["UploadURLRequest"];
export type UploadURL = components["schemas"]["UploadURL"];

// React Query key for a single recipe. Exported so the editor screen and the
// cache helpers below cannot drift apart.
export function recipeDetailKey(recipeId: string) {
  return ["recipe", recipeId] as const;
}

async function handleError(response: Response, fallback: string): Promise<never> {
  const body = await response.json().catch(() => null);
  throw new Error(body?.error || fallback);
}

// Resolves a URL returned by the API against the API origin. Absolute URLs pass
// through unchanged.
function toAbsoluteApiUrl(url: string): string {
  if (/^[a-z][a-z0-9+.-]*:\/\//i.test(url)) return url;
  return new URL(url, apiOrigin()).toString();
}

export async function listRecipes(params?: {
  search?: string;
  tags?: string[];
  difficulty?: string;
}): Promise<Recipe[]> {
  const q = new URLSearchParams();
  if (params?.search) q.set("search", params.search);
  if (params?.tags?.length) q.set("tags", params.tags.join(","));
  if (params?.difficulty) q.set("difficulty", params.difficulty);
  const url = `${config.apiBaseUrl}/recipes${q.toString() ? `?${q}` : ""}`;
  const res = await fetch(url);
  if (!res.ok) await handleError(res, `Failed to list recipes: ${res.status}`);
  return res.json() as Promise<Recipe[]>;
}

export async function getRecipe(recipeId: string): Promise<Recipe> {
  const res = await fetch(`${config.apiBaseUrl}/recipes/${recipeId}`);
  if (!res.ok) await handleError(res, `Failed to get recipe: ${res.status}`);
  return res.json() as Promise<Recipe>;
}

export async function createRecipe(req: CreateRecipeRequest): Promise<Recipe> {
  const res = await fetch(`${config.apiBaseUrl}/recipes`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to create recipe: ${res.status}`);
  return res.json() as Promise<Recipe>;
}

export async function updateRecipe(recipeId: string, req: CreateRecipeRequest): Promise<Recipe> {
  const res = await fetch(`${config.apiBaseUrl}/recipes/${recipeId}`, {
    method: "PUT",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to update recipe: ${res.status}`);
  return res.json() as Promise<Recipe>;
}

// Reconciles the React Query cache after a recipe was saved. Writing the server
// response into the detail entry is what makes a reopened recipe show what was
// just saved: without it the cache keeps the pre-save copy and the editor
// hydrates from stale values (e.g. an empty source_url).
export function applySavedRecipe(queryClient: QueryClient, saved: Recipe): void {
  queryClient.setQueryData(recipeDetailKey(saved.id), saved);
  // Mark the seeded entry stale as well: it renders immediately while the next
  // mount revalidates it against the server.
  queryClient.invalidateQueries({ queryKey: recipeDetailKey(saved.id) });
  queryClient.invalidateQueries({ queryKey: ["recipes"] });
}

export async function deleteRecipe(recipeId: string): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/recipes/${recipeId}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to delete recipe: ${res.status}`);
}

// Requests a presigned image upload URL. Requires the recipe to exist.
export async function generateRecipeUploadURL(
  recipeId: string,
  req: UploadURLRequest
): Promise<UploadURL> {
  const res = await fetch(`${config.apiBaseUrl}/recipes/${recipeId}/upload-url`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to get upload URL: ${res.status}`);
  const upload = (await res.json()) as UploadURL;
  return { ...upload, url: toAbsoluteApiUrl(upload.url) };
}

// Uploads a file to a presigned PUT URL.
export async function uploadToPresignedUrl(presignedUrl: string, file: File): Promise<void> {
  const res = await fetch(toAbsoluteApiUrl(presignedUrl), {
    method: "PUT",
    headers: { "Content-Type": file.type || "application/octet-stream" },
    body: file,
  });
  if (!res.ok) throw new Error(`Upload failed: ${res.status}`);
}

// Resolves a stored media value to a displayable URL. Full URLs pass through,
// object keys are prefixed with the media base. Returns null when empty.
export function resolveMediaUrl(value: string | null | undefined): string | null {
  if (!value) return null;
  if (/^https?:\/\//.test(value) || value.startsWith("data:")) return value;
  return `${mediaBaseUrl()}/${value.replace(/^\/+/, "")}`;
}
