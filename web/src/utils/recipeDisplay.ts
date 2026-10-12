import type { Recipe } from "../api/recipes";

/** Default recipe tags -> i18n label keys; custom tags display as-is. */
const TAG_LABEL_KEYS: Record<string, string> = {
  quick: "tags.quick",
  vegan: "tags.vegan",
  vegetarian: "tags.vegetarian",
  breakfast: "tags.breakfast",
  dinner: "tags.dinner",
  dessert: "tags.dessert",
  soup: "tags.soup",
  salad: "tags.salad",
};

/** Maps recipe tags to a food emoji used for the cover placeholder. */
const TAG_EMOJI: Record<string, string> = {
  breakfast: "🍳",
  dinner: "🥘",
  dessert: "🍰",
  vegan: "🥗",
  vegetarian: "🥦",
  soup: "🍜",
  salad: "🥗",
  quick: "⚡",
};

/** Badge classes used to colour the difficulty pill. */
const DIFFICULTY_BADGES: Record<string, string> = {
  easy: "badge-done",
  medium: "badge-habit",
  hard: "badge-todo",
};

/** Emoji shown on the cover placeholder for a recipe without a picture. */
const DEFAULT_EMOJI = "🍽️";

/** i18n key for a known tag, or null for a custom tag that is shown verbatim. */
export function tagLabelKey(tag: string): string | null {
  return TAG_LABEL_KEYS[tag] ?? null;
}

/** Translates a tag for display, falling back to the raw tag for custom ones. */
export function tagLabel(t: (key: string) => string, value: string): string {
  const key = tagLabelKey(value);
  return key ? t(key) : value;
}

/** Emoji used as the cover placeholder when the recipe has no image. */
export function recipeEmoji(recipe: Recipe): string {
  const tag = (recipe.tags ?? []).find((value) => TAG_EMOJI[value]);
  return tag ? TAG_EMOJI[tag] : DEFAULT_EMOJI;
}

/** Colour class for the difficulty badge shown on cards and in the modal. */
export function difficultyBadgeClass(difficulty: string): string {
  return DIFFICULTY_BADGES[difficulty] ?? "badge-todo";
}

/** Prep + cook time: the "total" shown on cards and in the modal stats. */
export function totalMinutes(recipe: Recipe): number {
  return recipe.prep_time_minutes + recipe.cook_time_minutes;
}

/** "200 g" from an ingredient row, or "" when the amount was left blank. */
export function ingredientAmount(ingredient: {
  quantity?: string | null;
  unit?: string | null;
}): string {
  return [ingredient.quantity, ingredient.unit]
    .map((part) => (part ?? "").trim())
    .filter(Boolean)
    .join(" ");
}
