import { describe, expect, it } from "vitest";
import type { Recipe } from "../api/recipes";
import { formValuesFromRecipe } from "./recipeForm";

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

describe("formValuesFromRecipe", () => {
  it("round-trips the saved source URL into the editor", () => {
    const values = formValuesFromRecipe(stored);

    expect(values.sourceUrl).toBe("https://www.youtube.com/watch?v=8ryJyIg0qIo&t=1s");
  });

  it("maps every persisted field, including the stored image key", () => {
    expect(formValuesFromRecipe(stored)).toEqual({
      title: "YouTube curry",
      description: "From a video",
      difficulty: "medium",
      prep: 10,
      cook: 25,
      servings: 2,
      tags: "dinner, quick",
      ingredients: [{ name: "Rice", quantity: "200", unit: "g" }],
      steps: [{ instruction: "Cook the rice" }],
      imageUrl: "abc/curry.jpg",
      sourceUrl: "https://www.youtube.com/watch?v=8ryJyIg0qIo&t=1s",
    });
  });

  it("treats a null source URL (cleared field) as empty", () => {
    expect(formValuesFromRecipe({ ...stored, source_url: null }).sourceUrl).toBe("");
  });

  it("falls back to empty values when the optional fields are missing", () => {
    const values = formValuesFromRecipe({
      ...stored,
      image_url: undefined,
      source_url: undefined,
      tags: undefined,
      ingredients: undefined,
      steps: undefined,
    });

    expect(values.imageUrl).toBe("");
    expect(values.sourceUrl).toBe("");
    expect(values.tags).toBe("");
    expect(values.ingredients).toEqual([]);
    expect(values.steps).toEqual([]);
  });
});
