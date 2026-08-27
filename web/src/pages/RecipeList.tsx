import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { deleteRecipe, listRecipes, resolveMediaUrl, type Recipe } from "../api/recipes";
import { useI18n } from "../i18n";
import { UtensilsIcon } from "../components/icons";

/** Maps the well-known default recipe tags to i18n label keys so they can be
 *  translated (e.g. "quick" → "Rápida" in pt-BR). Custom tags are shown as-is. */
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

function recipeEmoji(r: Recipe): string {
  const tag = (r.tags ?? []).find((t) => TAG_EMOJI[t]);
  return tag ? TAG_EMOJI[tag] : "🍽️";
}

export default function RecipeList({ onOpen }: { onOpen: (recipe: Recipe) => void }) {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [search, setSearch] = useState("");
  const [tag, setTag] = useState<string | null>(null);
  const [difficulty, setDifficulty] = useState<string | null>(null);

  const { data: recipes = [], isLoading } = useQuery({
    queryKey: ["recipes", search, tag, difficulty],
    queryFn: () =>
      listRecipes({ search: search || undefined, tags: tag ? [tag] : undefined, difficulty: difficulty || undefined }),
  });

  // Fetch all recipes (no filters) so the tag filter options follow the tags
  // that actually exist in the user's recipes instead of a pre-made list.
  const { data: allRecipes = [] } = useQuery({
    queryKey: ["recipes", "all"],
    queryFn: () => listRecipes(),
  });

  const availableTags = useMemo(() => {
    const seen = new Set<string>();
    for (const r of allRecipes) {
      for (const t2 of r.tags ?? []) seen.add(t2);
    }
    return [...seen].sort((a, b) => a.localeCompare(b));
  }, [allRecipes]);

  /** Translates a tag for display, falling back to the raw tag for custom ones. */
  const tagLabel = (value: string): string => {
    const key = TAG_LABEL_KEYS[value];
    return key ? t(key) : value;
  };

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteRecipe(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["recipes"] }),
  });

  return (
    <div className="animate-fade-in">
      <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-md)" }}>
        <h2 className="page-heading" style={{ marginBottom: 0 }}><UtensilsIcon size={24} /> {t("recipes.title")}</h2>
        <button className="btn btn-primary" onClick={() => onOpen({ id: "__new__" } as Recipe)}>
          {t("recipes.new")}
        </button>
      </div>

      {/* Filters */}
      <div style={{ display: "flex", gap: "var(--space-sm)", flexWrap: "wrap", marginBottom: "var(--space-md)" }}>
        <input
          className="input"
          placeholder={t("recipes.search")}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <select className="select" value={difficulty ?? ""} onChange={(e) => setDifficulty(e.target.value || null)}>
          <option value="">{t("recipes.allDifficulties")}</option>
          <option value="easy">{t("recipes.easy")}</option>
          <option value="medium">{t("recipes.medium")}</option>
          <option value="hard">{t("recipes.hard")}</option>
        </select>
      </div>

      <div style={{ display: "flex", gap: "0.4rem", flexWrap: "wrap", marginBottom: "var(--space-lg)" }}>
        {availableTags.map((t2) => (
          <button
            key={t2}
            className={`badge ${tag === t2 ? "badge-done" : "badge-habit"}`}
            style={{ cursor: "pointer", border: "none", fontFamily: "var(--font-family)" }}
            onClick={() => setTag(tag === t2 ? null : t2)}
          >
            #{tagLabel(t2)}
          </button>
        ))}
      </div>

      {isLoading && <p style={{ color: "var(--color-text-secondary)" }}>{t("recipes.loading")}</p>}

      {recipes.length === 0 && !isLoading && (
        <div className="empty-state">
          <div className="empty-state-icon">🍳</div>
          <p className="empty-state-text">{t("recipes.empty")}</p>
        </div>
      )}

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(220px, 1fr))", gap: "var(--space-md)" }}>
        {recipes.map((r) => (
          <div key={r.id} className="card" style={{ cursor: "pointer" }} onClick={() => onOpen(r)}>
            {resolveMediaUrl(r.image_url) ? (
              <img src={resolveMediaUrl(r.image_url)!} alt={r.title} className="recipe-thumb" loading="lazy" />
            ) : (
              <div className="recipe-thumb-placeholder" aria-hidden="true">
                {recipeEmoji(r)}
              </div>
            )}
            <div className="flex-center" style={{ justifyContent: "space-between" }}>
              <span className="card-title">{r.title}</span>
              <span className={`badge ${r.difficulty === "easy" ? "badge-done" : r.difficulty === "medium" ? "badge-habit" : "badge-todo"}`}>
                {t(`recipes.${r.difficulty}`)}
              </span>
            </div>
            {r.description && <p style={{ color: "var(--color-text-secondary)", fontSize: "var(--font-size-sm)", margin: "0.35rem 0" }}>{r.description}</p>}
            <p style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
              ⏱ {t("recipes.meta", { minutes: r.prep_time_minutes + r.cook_time_minutes, servings: r.servings })}
              {r.tags?.length ? ` · ${r.tags.map((t2) => `#${tagLabel(t2)}`).join(" ")}` : ""}
            </p>
            <button
              className="btn btn-danger btn-sm"
              onClick={(e) => {
                e.stopPropagation();
                deleteMut.mutate(r.id);
              }}
            >
              {t("common.delete")}
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
