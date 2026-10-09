import type { Meal, MealPlanDay, MealPlanItem } from "../api/mealplan";
import { DAY_OPTIONS } from "./constants";

/** Meal slots, top to bottom in the weekly grid. */
export const MEAL_ORDER: Meal[] = ["breakfast", "lunch", "dinner", "snack"];

/** Seven day columns, Monday first. */
export const MEAL_PLAN_DAYS: MealPlanDay[] = DAY_OPTIONS.map((d) => d.value);

/** i18n key for a meal slot label. */
export function mealLabelKey(meal: Meal): string {
  return `mealPlan.meal.${meal}`;
}

/** i18n key for a weekday label. */
export function dayLabelKey(day: MealPlanDay): string {
  return `days.${day}`;
}

/** Items sitting on one day and one meal slot, in insertion order. */
export function itemsForCell(items: MealPlanItem[], day: MealPlanDay, meal: Meal): MealPlanItem[] {
  return items.filter((item) => item.day === day && item.meal === meal);
}

/** Sums the calories of every item matching the optional day/meal filter. */
export function totalCalories(items: MealPlanItem[], day?: MealPlanDay, meal?: Meal): number {
  return items.reduce((sum, item) => {
    if (day && item.day !== day) return sum;
    if (meal && item.meal !== meal) return sum;
    return sum + item.calories;
  }, 0);
}

/** Calories per day column, keyed by weekday. Days without items are 0. */
export function caloriesByDay(items: MealPlanItem[]): Record<MealPlanDay, number> {
  const totals: Record<MealPlanDay, number> = {
    mon: 0,
    tue: 0,
    wed: 0,
    thu: 0,
    fri: 0,
    sat: 0,
    sun: 0,
  };
  for (const item of items) {
    totals[item.day] += item.calories;
  }
  return totals;
}

/** Calories per meal slot, keyed by meal. Slots without items are 0. */
export function caloriesByMeal(items: MealPlanItem[]): Record<Meal, number> {
  const totals: Record<Meal, number> = {
    breakfast: 0,
    lunch: 0,
    dinner: 0,
    snack: 0,
  };
  for (const item of items) {
    totals[item.meal] += item.calories;
  }
  return totals;
}

/** Number of automatic chip tones; keep in sync with the --tone-* tokens. */
export const MEAL_PLAN_TONES = 6;

/** Stable tone (1-based) for a food name, so a dish keeps its colour everywhere. */
export function toneForName(name: string): number {
  const key = name.trim().toLowerCase();
  let hash = 0;
  for (let i = 0; i < key.length; i += 1) {
    hash = (hash * 31 + key.charCodeAt(i)) % 1000003;
  }
  return (hash % MEAL_PLAN_TONES) + 1;
}

/** Chip class selecting that tone. */
export function toneClass(name: string): string {
  return `tone-${toneForName(name)}`;
}
