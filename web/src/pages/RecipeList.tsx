import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { deleteRecipe, listRecipes, resolveMediaUrl } from "../api/recipes";
import { useI18n } from "../i18n";
import { UtensilsIcon } from "../components/icons";
import RecipeViewModal from "../components/RecipeViewModal";
import { difficultyBadgeClass, recipeEmoji, tagLabelKey, totalMinutes } from "../utils/recipeDisplay";

interface RecipeListProps {
  onNew: () => void;
  onEdit: (recipeId: string) => void;
}

export default function RecipeList({ onNew, onEdit }: RecipeListProps) {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [search, setSearch] = useState("");
  const [tag, setTag] = useState<string | null>(null);
  const [difficulty, setDifficulty] = useState<string | null>(null);
  // Recipe whose read-only detail modal is open (null = none).
  const [viewingId, setViewingId] = useState<string | null>(null);

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
    const key = tagLabelKey(value);
    return key ? t(key) : value;
  };

  const deleteMut = useMutation({
    mutationFn: (id: string) => deleteRecipe(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recipes"] });
      // The modal can be showing the recipe that was just deleted.
      setViewingId(null);
    },
  });

  return (
    <div className="animate-fade-in">
      <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-md)" }}>
        <h2 className="page-heading" style={{ marginBottom: 0 }}><UtensilsIcon size={24} /> {t("recipes.title")}</h2>
        <button className="btn btn-primary" onClick={onNew}>
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
          <div
            key={r.id}
            className="card card-interactive"
            role="button"
            tabIndex={0}
            aria-label={t("recipes.openDetails", { title: r.title })}
            onClick={() => setViewingId(r.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                setViewingId(r.id);
              }
            }}
          >
            {resolveMediaUrl(r.image_url) ? (
              <img src={resolveMediaUrl(r.image_url)!} alt={r.title} className="recipe-thumb" loading="lazy" />
            ) : (
              <div className="recipe-thumb-placeholder" aria-hidden="true">
                {recipeEmoji(r)}
              </div>
            )}
            <div className="flex-center" style={{ justifyContent: "space-between" }}>
              <span className="card-title">{r.title}</span>
              <span className={`badge ${difficultyBadgeClass(r.difficulty)}`}>
                {t(`recipes.${r.difficulty}`)}
              </span>
            </div>
            {r.description && <p style={{ color: "var(--color-text-secondary)", fontSize: "var(--font-size-sm)", margin: "0.35rem 0" }}>{r.description}</p>}
            <p style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
              ⏱ {t("recipes.meta", { minutes: totalMinutes(r), servings: r.servings })}
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

      {viewingId && (
        <RecipeViewModal
          recipeId={viewingId}
          onClose={() => setViewingId(null)}
          onEdit={(id) => {
            setViewingId(null);
            onEdit(id);
          }}
        />
      )}
    </div>
  );
}
