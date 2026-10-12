import type { QueryClient } from "@tanstack/react-query";
import { apiOrigin, mediaBaseUrl } from "../config";
import { request } from "./client";
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
  return request<Recipe[]>(`/recipes${q.toString() ? `?${q}` : ""}`, { action: "list recipes" });
}

export async function getRecipe(recipeId: string): Promise<Recipe> {
  return request<Recipe>(`/recipes/${recipeId}`, { action: "get recipe" });
}

export async function createRecipe(req: CreateRecipeRequest): Promise<Recipe> {
  return request<Recipe>("/recipes", {
    method: "POST",
    body: JSON.stringify(req),
    action: "create recipe",
  });
}

export async function updateRecipe(recipeId: string, req: CreateRecipeRequest): Promise<Recipe> {
  return request<Recipe>(`/recipes/${recipeId}`, {
    method: "PUT",
    body: JSON.stringify(req),
    action: "update recipe",
  });
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
  return request<void>(`/recipes/${recipeId}`, { method: "DELETE", action: "delete recipe" });
}

// Requests a presigned image upload URL. Requires the recipe to exist.
export async function generateRecipeUploadURL(
  recipeId: string,
  req: UploadURLRequest,
): Promise<UploadURL> {
  const upload = await request<UploadURL>(`/recipes/${recipeId}/upload-url`, {
    method: "POST",
    body: JSON.stringify(req),
    action: "get upload URL",
  });
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
