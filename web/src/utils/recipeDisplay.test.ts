import { describe, expect, it } from "vitest";
import type { Recipe } from "../api/recipes";
import {
  difficultyBadgeClass,
  ingredientAmount,
  recipeEmoji,
  tagLabelKey,
  totalMinutes,
} from "./recipeDisplay";

// Same shape as a recipe returned by GET /api/v1/recipes/{id}.
const stored: Recipe = {
  id: "2b1f0f9e-8f4f-4d3a-9c5b-2f0a3d7c1e55",
  title: "YouTube curry",
  description: "From a video",
  difficulty: "medium",
  prep_time_minutes: 10,
  cook_time_minutes: 25,
  servings: 2,
  image_url: "abc/curry.jpg",
  source_url: "https://www.youtube.com/watch?v=8ryJyIg0qIo&t=1s",
  tags: ["dinner", "quick"],
  ingredients: [{ id: "i1", name: "Rice", quantity: "200", unit: "g", sort_order: 0 }],
  steps: [{ id: "s1", step_number: 1, instruction: "Cook the rice" }],
  created_at: "2026-09-29T14:13:24Z",
  updated_at: "2026-09-29T14:13:24Z",
};

describe("recipeEmoji", () => {
  it("uses the emoji of the first recognised tag", () => {
    expect(recipeEmoji(stored)).toBe("🥘");
  });

  it("ignores custom tags and falls back to the generic cover", () => {
    expect(recipeEmoji({ ...stored, tags: ["grandma", "spicy"] })).toBe("🍽️");
  });

  it("falls back to the generic cover when the recipe has no tags", () => {
    expect(recipeEmoji({ ...stored, tags: undefined })).toBe("🍽️");
  });
});

describe("tagLabelKey", () => {
  it("maps the default tags to their i18n keys", () => {
    expect(tagLabelKey("breakfast")).toBe("tags.breakfast");
  });

  it("returns null for custom tags so they are shown verbatim", () => {
    expect(tagLabelKey("grandma")).toBeNull();
  });
});

describe("difficultyBadgeClass", () => {
  it("colours every known difficulty", () => {
    expect(difficultyBadgeClass("easy")).toBe("badge-done");
    expect(difficultyBadgeClass("medium")).toBe("badge-habit");
    expect(difficultyBadgeClass("hard")).toBe("badge-todo");
  });

  it("falls back to the todo badge for an unknown difficulty", () => {
    expect(difficultyBadgeClass("impossible")).toBe("badge-todo");
  });
});

describe("totalMinutes", () => {
  it("adds prep and cook time", () => {
    expect(totalMinutes(stored)).toBe(35);
  });
});

describe("ingredientAmount", () => {
  it("joins quantity and unit", () => {
    expect(ingredientAmount({ quantity: "200", unit: "g" })).toBe("200 g");
  });

  it("keeps whichever half is present", () => {
    expect(ingredientAmount({ quantity: "2", unit: null })).toBe("2");
    expect(ingredientAmount({ quantity: null, unit: "g" })).toBe("g");
  });

  it("returns an empty string when the amount was left blank", () => {
    expect(ingredientAmount({})).toBe("");
    expect(ingredientAmount({ quantity: "  ", unit: null })).toBe("");
  });
});
