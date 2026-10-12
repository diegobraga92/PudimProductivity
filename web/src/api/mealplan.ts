import { request } from "./client";
import type { components } from "./generated/mealplan-v1";

// Types are generated from api/openapi/mealplan-v1.yaml (the source of truth).
export type MealPlanItem = components["schemas"]["MealPlanItem"];
export type MealPlanDay = components["schemas"]["Day"];
export type Meal = components["schemas"]["Meal"];
export type CreateMealPlanItemRequest = components["schemas"]["CreateMealPlanItemRequest"];
export type UpdateMealPlanItemRequest = components["schemas"]["UpdateMealPlanItemRequest"];

// React Query key for the whole template, so invalidation and the fetch cannot drift apart.
export const MEAL_PLAN_KEY = ["mealPlan"] as const;

export async function listMealPlanItems(): Promise<MealPlanItem[]> {
  return request<MealPlanItem[]>("/meal-plan", { action: "load meal plan" });
}

/** Returns one item per selected day. */
export async function createMealPlanItem(
  req: CreateMealPlanItemRequest,
): Promise<MealPlanItem[]> {
  return request<MealPlanItem[]>("/meal-plan", {
    method: "POST",
    body: JSON.stringify(req),
    action: "add meal",
  });
}

export async function updateMealPlanItem(
  itemId: string,
  req: UpdateMealPlanItemRequest,
): Promise<MealPlanItem> {
  return request<MealPlanItem>(`/meal-plan/${itemId}`, {
    method: "PUT",
    body: JSON.stringify(req),
    action: "update meal",
  });
}

export async function deleteMealPlanItem(itemId: string): Promise<void> {
  return request<void>(`/meal-plan/${itemId}`, { method: "DELETE", action: "delete meal" });
}

export async function clearMealPlanItems(): Promise<void> {
  return request<void>("/meal-plan", { method: "DELETE", action: "clear meal plan" });
}
