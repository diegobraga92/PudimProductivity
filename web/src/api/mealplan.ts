import config from "../config";
import { apiHeaders } from "./client";
import type { components } from "./generated/mealplan-v1";

// Types are generated from api/openapi/mealplan-v1.yaml (the source of truth).
export type MealPlanItem = components["schemas"]["MealPlanItem"];
export type MealPlanDay = components["schemas"]["Day"];
export type Meal = components["schemas"]["Meal"];
export type CreateMealPlanItemRequest = components["schemas"]["CreateMealPlanItemRequest"];
export type UpdateMealPlanItemRequest = components["schemas"]["UpdateMealPlanItemRequest"];

// React Query key for the whole template. The page reads it with this exact key
// so live invalidation and the fetch cannot drift apart.
export const MEAL_PLAN_KEY = ["mealPlan"] as const;

async function handleError(response: Response, fallback: string): Promise<never> {
  const body = await response.json().catch(() => null);
  throw new Error(body?.error || fallback);
}

export async function listMealPlanItems(): Promise<MealPlanItem[]> {
  const res = await fetch(`${config.apiBaseUrl}/meal-plan`);
  if (!res.ok) await handleError(res, `Failed to load meal plan: ${res.status}`);
  return res.json() as Promise<MealPlanItem[]>;
}

// Returns one item per selected day.
export async function createMealPlanItem(req: CreateMealPlanItemRequest): Promise<MealPlanItem[]> {
  const res = await fetch(`${config.apiBaseUrl}/meal-plan`, {
    method: "POST",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to add meal: ${res.status}`);
  return res.json() as Promise<MealPlanItem[]>;
}

export async function updateMealPlanItem(
  itemId: string,
  req: UpdateMealPlanItemRequest
): Promise<MealPlanItem> {
  const res = await fetch(`${config.apiBaseUrl}/meal-plan/${itemId}`, {
    method: "PUT",
    headers: apiHeaders(),
    body: JSON.stringify(req),
  });
  if (!res.ok) await handleError(res, `Failed to update meal: ${res.status}`);
  return res.json() as Promise<MealPlanItem>;
}

export async function deleteMealPlanItem(itemId: string): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/meal-plan/${itemId}`, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to delete meal: ${res.status}`);
}

export async function clearMealPlanItems(): Promise<void> {
  const res = await fetch(`${config.apiBaseUrl}/meal-plan`, {
    method: "DELETE",
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to clear meal plan: ${res.status}`);
}
