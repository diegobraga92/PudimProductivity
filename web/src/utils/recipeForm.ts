import type { Recipe } from "../api/recipes";

export type RecipeFormValues = {
  title: string;
  description: string;
  difficulty: "easy" | "medium" | "hard";
  prep: number;
  cook: number;
  servings: number;
  tags: string;
  ingredients: { name: string; quantity: string; unit: string }[];
  steps: { instruction: string }[];
  imageUrl: string;
  sourceUrl: string;
};

// API recipe -> editor form state. Every savable field must round-trip here, or reopening looks unpersisted.
export function formValuesFromRecipe(recipe: Recipe): RecipeFormValues {
  return {
    title: recipe.title,
    description: recipe.description ?? "",
    difficulty: recipe.difficulty,
    prep: recipe.prep_time_minutes,
    cook: recipe.cook_time_minutes,
    servings: recipe.servings,
    tags: (recipe.tags ?? []).join(", "),
    ingredients: recipe.ingredients?.length
      ? recipe.ingredients.map((i) => ({
          name: i.name,
          quantity: i.quantity ?? "",
          unit: i.unit ?? "",
        }))
      : [],
    steps: recipe.steps?.length
      ? recipe.steps.map((s) => ({ instruction: s.instruction }))
      : [],
    imageUrl: recipe.image_url ?? "",
    sourceUrl: recipe.source_url ?? "",
  };
}
