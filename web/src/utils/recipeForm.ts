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

// Maps a recipe fetched from the API onto the editor's form state. Every field
// the editor can save has to round-trip through here: a field missing from this
// mapping looks like it was never persisted when the recipe is reopened.
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
