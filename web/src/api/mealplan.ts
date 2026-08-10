import config from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/mealplan-v1";

// Types are generated from api/openapi/mealplan-v1.yaml.
export type MealPlan = components["schemas"]["MealPlan"];
export type CreateMealPlanRequest = components["schemas"]["CreateMealPlanRequest"];
export type ShoppingItem = components["schemas"]["ShoppingItem"];
export type MealType = components["schemas"]["MealType"];

async function handleError(response: Response, fallback: string): Promise<never> {
  const body = await response.json().catch(() => null);
  throw new Error(body?.error || fallback);
}

export async function listMealPlans(): Promise<MealPlan[]> {
  const res = await fetch(`${config.apiBaseUrl}/mealplans`);
  if (!res.ok) await handleError(res, `Failed to list meal plans: ${res.status}`);
  return res.json() as Promise<MealPlan[]>;
}

export async function getMealPlan(planId: string): Promise<MealPlan> {
  const res = await fetch(`${config.apiBaseUrl}/mealplans/${planId}`);
  if (!res.ok) await handleError(res, `Failed to get meal plan: ${res.status}`);
  return res.json() as Promise<MealPlan>;
}

export async function createMealPlan(req: CreateMealPlanRequest): Promise<MealPlan> {
  const res = await fetch(`${config.apiBaseUrl}/mealplans`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to create meal plan: ${res.status}`);
  return res.json() as Promise<MealPlan>;
}

export async function deleteMealPlan(planId: string): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/mealplans/${planId}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to delete meal plan: ${res.status}`);
}

export async function generateShoppingList(planId: string): Promise<ShoppingItem[]> {
  const res = await fetch(`${config.apiBaseUrl}/mealplans/${planId}/shopping-list`, {
    method: "POST",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to generate shopping list: ${res.status}`);
  return res.json() as Promise<ShoppingItem[]>;
}

export async function getShoppingList(planId: string): Promise<ShoppingItem[]> {
  const res = await fetch(`${config.apiBaseUrl}/mealplans/${planId}/shopping-list`);
  if (!res.ok) await handleError(res, `Failed to get shopping list: ${res.status}`);
  return res.json() as Promise<ShoppingItem[]>;
}

export async function toggleShoppingItem(planId: string, itemId: string): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/mealplans/${planId}/shopping-list/${itemId}`, {
    method: "PUT",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to toggle item: ${res.status}`);
}

export async function publishMealPlan(planId: string): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/mealplans/${planId}/publish`, {
    method: "POST",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to publish meal plan: ${res.status}`);
}
