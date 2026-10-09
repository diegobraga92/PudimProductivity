import { describe, expect, it } from "vitest";
import type { MealPlanItem } from "../api/mealplan";
import {
  caloriesByDay,
  caloriesByMeal,
  itemsForCell,
  MEAL_ORDER,
  MEAL_PLAN_DAYS,
  mealLabelKey,
  totalCalories,
} from "./mealPlan";

function item(partial: Pick<MealPlanItem, "id" | "day" | "meal"> & Partial<MealPlanItem>): MealPlanItem {
  return {
    name: "Food",
    amount: "",
    calories: 0,
    recipe_id: null,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    ...partial,
  };
}

const plan: MealPlanItem[] = [
  item({ id: "1", day: "mon", meal: "breakfast", name: "Oats", calories: 300 }),
  item({ id: "2", day: "mon", meal: "lunch", name: "Rice", calories: 600 }),
  item({ id: "3", day: "wed", meal: "breakfast", name: "Oats", calories: 300 }),
];

describe("meal plan shape", () => {
  it("lists the four slots and seven days in grid order", () => {
    expect(MEAL_ORDER).toEqual(["breakfast", "lunch", "dinner", "snack"]);
    expect(MEAL_PLAN_DAYS).toEqual(["mon", "tue", "wed", "thu", "fri", "sat", "sun"]);
  });

  it("maps a slot to its i18n key", () => {
    expect(mealLabelKey("dinner")).toBe("mealPlan.meal.dinner");
  });
});

describe("itemsForCell", () => {
  it("keeps only the items of one day and slot", () => {
    expect(itemsForCell(plan, "mon", "breakfast").map((i) => i.id)).toEqual(["1"]);
    expect(itemsForCell(plan, "mon", "dinner")).toEqual([]);
  });
});

describe("totalCalories", () => {
  it("adds every item when no filter is given", () => {
    expect(totalCalories(plan)).toBe(1200);
  });

  it("narrows down to a day or a slot", () => {
    expect(totalCalories(plan, "mon")).toBe(900);
    expect(totalCalories(plan, undefined, "breakfast")).toBe(600);
  });

  it("returns 0 for an empty plan", () => {
    expect(totalCalories([])).toBe(0);
  });
});

describe("caloriesByDay", () => {
  it("zero-fills every day of the week", () => {
    const totals = caloriesByDay(plan);
    expect(totals.mon).toBe(900);
    expect(totals.wed).toBe(300);
    expect(totals.tue).toBe(0);
    expect(totals.sun).toBe(0);
    expect(Object.keys(totals)).toEqual(MEAL_PLAN_DAYS);
  });
});

describe("caloriesByMeal", () => {
  it("zero-fills every slot", () => {
    const totals = caloriesByMeal(plan);
    expect(totals.breakfast).toBe(600);
    expect(totals.lunch).toBe(600);
    expect(totals.dinner).toBe(0);
    expect(totals.snack).toBe(0);
    expect(Object.keys(totals)).toEqual(MEAL_ORDER);
  });
});
