import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { deleteRecipe, getRecipe, recipeDetailKey, resolveMediaUrl } from "../api/recipes";
import { useI18n } from "../i18n";
import {
  difficultyBadgeClass,
  ingredientAmount,
  recipeEmoji,
  tagLabelKey,
  totalMinutes,
} from "../utils/recipeDisplay";
import Modal from "./Modal";
import { useConfirm } from "./useConfirm";

interface RecipeViewModalProps {
  recipeId: string;
  onClose: () => void;
  onEdit: (recipeId: string) => void;
}

/**
 * Read-only recipe detail, opened by clicking a recipe card. Ingredients and
 * steps are not part of the list payload, so the full recipe is fetched with
 * the same query key the editor hydrates from - opening the modal therefore
 * warms the editor's cache. Editing and deleting both leave from here.
 */
export default function RecipeViewModal({ recipeId, onClose, onEdit }: RecipeViewModalProps) {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const confirm = useConfirm();
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const { data: recipe, isLoading, isError, refetch } = useQuery({
    queryKey: recipeDetailKey(recipeId),
    queryFn: () => getRecipe(recipeId),
  });

  const deleteMut = useMutation({
    mutationFn: () => deleteRecipe(recipeId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recipes"] });
      onClose();
    },
    onError: (err: Error) => setDeleteError(err.message),
  });

  /** Translates a tag for display, falling back to the raw tag for custom ones. */
  const tagLabel = (value: string): string => {
    const key = tagLabelKey(value);
    return key ? t(key) : value;
  };

  async function handleDelete() {
    const confirmed = await confirm({
      title: t("recipes.deleteTitle"),
      message: t("library.cannotUndo"),
      confirmLabel: t("common.delete"),
      confirmVariant: "danger",
    });
    if (confirmed) {
      setDeleteError(null);
      deleteMut.mutate();
    }
  }

  return (
    <Modal onClose={onClose} maxWidth={640} ariaLabel={t("recipes.detailsTitle")}>
      <h3 className="modal-title">{t("recipes.detailsTitle")}</h3>

      {isLoading && <p className="text-sm text-secondary">{t("common.loading")}</p>}

      {isError && (
        <div style={{ marginTop: "var(--space-sm)" }}>
          <p className="error-text text-sm">{t("common.error")}</p>
          <button className="btn btn-sm" style={{ marginTop: "var(--space-sm)" }} onClick={() => refetch()}>
            {t("common.retry")}
          </button>
        </div>
      )}

      {recipe && (
        <div>
          {resolveMediaUrl(recipe.image_url) ? (
            <img
              src={resolveMediaUrl(recipe.image_url)!}
              alt={t("recipes.previewAlt")}
              className="recipe-modal-hero"
            />
          ) : (
            <div className="recipe-modal-hero recipe-modal-hero-placeholder" aria-hidden="true">
              {recipeEmoji(recipe)}
            </div>
          )}

          <div className="flex-center" style={{ justifyContent: "space-between", gap: "var(--space-sm)" }}>
            <h4 className="recipe-modal-title">{recipe.title}</h4>
            <span className={`badge ${difficultyBadgeClass(recipe.difficulty)}`}>
              {t(`recipes.${recipe.difficulty}`)}
            </span>
          </div>

          <div className="recipe-modal-stats">
            <div>
              <span className="recipe-stat-label">{t("recipes.prepTime")}</span>
              <span className="recipe-stat-value">
                {t("recipes.minutes", { minutes: recipe.prep_time_minutes })}
              </span>
            </div>
            <div>
              <span className="recipe-stat-label">{t("recipes.cookTime")}</span>
              <span className="recipe-stat-value">
                {t("recipes.minutes", { minutes: recipe.cook_time_minutes })}
              </span>
            </div>
            <div>
              <span className="recipe-stat-label">{t("recipes.totalTime")}</span>
              <span className="recipe-stat-value">
                {t("recipes.minutes", { minutes: totalMinutes(recipe) })}
              </span>
            </div>
            <div>
              <span className="recipe-stat-label">{t("recipes.servings")}</span>
              <span className="recipe-stat-value">{recipe.servings}</span>
            </div>
          </div>

          {recipe.description && (
            <p className="text-sm" style={{ margin: "0 0 var(--space-sm)" }}>{recipe.description}</p>
          )}

          {recipe.tags?.length ? (
            <div style={{ display: "flex", gap: "0.4rem", flexWrap: "wrap" }}>
              {recipe.tags.map((tag) => (
                <span key={tag} className="badge badge-habit">#{tagLabel(tag)}</span>
              ))}
            </div>
          ) : null}

          <span className="recipe-section-title">{t("recipes.ingredients")}</span>
          {recipe.ingredients?.length ? (
            recipe.ingredients.map((ingredient) => (
              <div key={ingredient.id ?? ingredient.name} className="recipe-ingredient-row">
                <span className="recipe-ingredient-amount">{ingredientAmount(ingredient)}</span>
                <span>{ingredient.name}</span>
              </div>
            ))
          ) : (
            <p className="text-sm text-secondary" style={{ margin: 0 }}>{t("recipes.noIngredients")}</p>
          )}

          <span className="recipe-section-title">{t("recipes.steps")}</span>
          {recipe.steps?.length ? (
            recipe.steps.map((step, index) => (
              <div key={step.id ?? index} className="recipe-step-row">
                <span className="recipe-step-number">{step.step_number ?? index + 1}.</span>
                <span>{step.instruction}</span>
              </div>
            ))
          ) : (
            <p className="text-sm text-secondary" style={{ margin: 0 }}>{t("recipes.noSteps")}</p>
          )}

          {recipe.source_url && (
            <a
              className="recipe-source-link"
              href={recipe.source_url}
              target="_blank"
              rel="noreferrer"
              style={{ marginTop: "var(--space-md)" }}
            >
              🔗 {t("recipes.viewSource")}
            </a>
          )}

          {deleteError && (
            <p className="text-sm" style={{ color: "var(--color-danger)", marginTop: "var(--space-sm)" }}>
              {deleteError}
            </p>
          )}

          <div className="modal-actions" style={{ justifyContent: "space-between", marginTop: "var(--space-lg)" }}>
            <button className="btn btn-danger btn-sm" disabled={deleteMut.isPending} onClick={handleDelete}>
              {deleteMut.isPending ? t("common.deleting") : t("common.delete")}
            </button>
            <div style={{ display: "flex", gap: "var(--space-sm)" }}>
              <button className="btn btn-sm" onClick={onClose}>{t("common.close")}</button>
              <button className="btn btn-primary btn-sm" onClick={() => onEdit(recipe.id)}>{t("common.edit")}</button>
            </div>
          </div>
        </div>
      )}
    </Modal>
  );
}
