import config from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/recipes-v1";

// Types are generated from api/openapi/recipes-v1.yaml (the source of truth).
export type Recipe = components["schemas"]["Recipe"];
export type CreateRecipeRequest = components["schemas"]["CreateRecipeRequest"];

async function handleError(response: Response, fallback: string): Promise<never> {
  const body = await response.json().catch(() => null);
  throw new Error(body?.error || fallback);
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

export async function deleteRecipe(recipeId: string): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/recipes/${recipeId}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to delete recipe: ${res.status}`);
}
