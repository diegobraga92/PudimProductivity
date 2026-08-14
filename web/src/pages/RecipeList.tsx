import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { deleteRecipe, listRecipes, resolveMediaUrl, type Recipe } from "../api/recipes";

const ALL_TAGS = ["quick", "vegan", "vegetarian", "breakfast", "dinner", "dessert", "soup", "salad"];

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
  const [search, setSearch] = useState("");
  const [tag, setTag] = useState<string | null>(null);
  const [difficulty, setDifficulty] = useState<string | null>(null);

  const { data: recipes = [], isLoading } = useQuery({
    queryKey: ["recipes", search, tag, difficulty],
    queryFn: () =>
      listRecipes({ search: search || undefined, tags: tag ? [tag] : undefined, difficulty: difficulty || undefined }),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteRecipe(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["recipes"] }),
  });

  return (
    <div className="animate-fade-in">
      <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-md)" }}>
        <h2 className="page-heading" style={{ marginBottom: 0 }}>🍳 Recipes</h2>
        <button className="btn btn-primary" onClick={() => onOpen({ id: "__new__" } as Recipe)}>
          + New recipe
        </button>
      </div>

      {/* Filters */}
      <div style={{ display: "flex", gap: "var(--space-sm)", flexWrap: "wrap", marginBottom: "var(--space-md)" }}>
        <input
          className="input"
          placeholder="Search recipes…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <select className="select" value={difficulty ?? ""} onChange={(e) => setDifficulty(e.target.value || null)}>
          <option value="">All difficulties</option>
          <option value="easy">Easy</option>
          <option value="medium">Medium</option>
          <option value="hard">Hard</option>
        </select>
      </div>

      <div style={{ display: "flex", gap: "0.4rem", flexWrap: "wrap", marginBottom: "var(--space-lg)" }}>
        {ALL_TAGS.map((t) => (
          <button
            key={t}
            className={`badge ${tag === t ? "badge-done" : "badge-habit"}`}
            style={{ cursor: "pointer", border: "none", fontFamily: "var(--font-family)" }}
            onClick={() => setTag(tag === t ? null : t)}
          >
            #{t}
          </button>
        ))}
      </div>

      {isLoading && <p style={{ color: "var(--color-text-secondary)" }}>Loading recipes…</p>}

      {recipes.length === 0 && !isLoading && (
        <div className="empty-state">
          <div className="empty-state-icon">🍳</div>
          <p className="empty-state-text">No recipes yet. Create your first one!</p>
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
                {r.difficulty}
              </span>
            </div>
            {r.description && <p style={{ color: "var(--color-text-secondary)", fontSize: "var(--font-size-sm)", margin: "0.35rem 0" }}>{r.description}</p>}
            <p style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
              ⏱ {r.prep_time_minutes + r.cook_time_minutes} min · 🍽 {r.servings} serv
              {r.tags?.length ? ` · ${r.tags.map((t) => `#${t}`).join(" ")}` : ""}
            </p>
            <button
              className="btn btn-danger btn-sm"
              onClick={(e) => {
                e.stopPropagation();
                deleteMut.mutate(r.id);
              }}
            >
              Delete
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
