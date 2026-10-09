import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Fragment, useMemo, useState, type FormEvent } from "react";
import {
  createMealPlanItem,
  deleteMealPlanItem,
  listMealPlanItems,
  MEAL_PLAN_KEY,
  updateMealPlanItem,
  type Meal,
  type MealPlanDay,
  type MealPlanItem,
} from "../api/mealplan";
import { listRecipes } from "../api/recipes";
import { MealPlanIcon } from "../components/icons";
import Modal from "../components/Modal";
import { useConfirm } from "../components/useConfirm";
import { useI18n } from "../i18n";
import {
  caloriesByDay,
  caloriesByMeal,
  dayLabelKey,
  itemsForCell,
  MEAL_ORDER,
  MEAL_PLAN_DAYS,
  mealLabelKey,
  totalCalories,
} from "../utils/mealPlan";

/** What the editor modal is working on: a brand-new cell or an existing item. */
type EditorTarget =
  | { mode: "create"; day: MealPlanDay; meal: Meal }
  | { mode: "edit"; item: MealPlanItem };

export default function MealPlanner() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const confirm = useConfirm();
  const [editor, setEditor] = useState<EditorTarget | null>(null);
  const [error, setError] = useState<string | null>(null);

  const { data: items = [], isLoading } = useQuery({
    queryKey: MEAL_PLAN_KEY,
    queryFn: listMealPlanItems,
  });

  // Recipes back the optional link picker and name the linked chips. The list
  // payload is enough: titles are all the chips need.
  const { data: recipes = [] } = useQuery({
    queryKey: ["recipes", "all"],
    queryFn: () => listRecipes(),
  });

  const recipeTitles = useMemo(() => new Map(recipes.map((r) => [r.id, r.title])), [recipes]);
  const dayTotals = caloriesByDay(items);
  const mealTotals = caloriesByMeal(items);
  const weeklyTotal = totalCalories(items);

  const deleteMut = useMutation({
    mutationFn: (itemId: string) => deleteMealPlanItem(itemId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: MEAL_PLAN_KEY }),
    onError: (err: Error) => setError(err.message),
  });

  async function handleDelete(item: MealPlanItem) {
    const confirmed = await confirm({
      title: t("mealPlan.deleteTitle"),
      message: t("mealPlan.deleteMessage", { name: item.name }),
      confirmLabel: t("common.delete"),
      confirmVariant: "danger",
    });
    if (confirmed) {
      setError(null);
      deleteMut.mutate(item.id);
    }
  }

  return (
    <div className="animate-fade-in">
      <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-sm)" }}>
        <h2 className="page-heading" style={{ marginBottom: 0 }}>
          <MealPlanIcon size={24} /> {t("mealPlan.title")}
        </h2>
        <span className="meal-plan-total-chip">
          {t("mealPlan.weeklyTotal")}: {t("mealPlan.kcal", { calories: weeklyTotal })}
        </span>
      </div>

      <p className="text-sm text-secondary" style={{ marginTop: 0, marginBottom: "var(--space-lg)" }}>
        {t("mealPlan.subtitle")}
      </p>

      {error && <p className="error-text text-sm">{error}</p>}

      {isLoading && <p className="text-sm text-secondary">{t("mealPlan.loading")}</p>}

      {!isLoading && items.length === 0 && (
        <div className="empty-state" style={{ marginBottom: "var(--space-md)" }}>
          <div className="empty-state-icon">🥗</div>
          <p className="empty-state-text">{t("mealPlan.empty")}</p>
        </div>
      )}

      <div className="meal-plan-scroll">
        <div className="meal-plan-grid">
          <div className="meal-plan-corner" />
          {MEAL_PLAN_DAYS.map((day) => (
            <div key={day} className="meal-plan-dayhead">
              {t(dayLabelKey(day))}
            </div>
          ))}
          <div className="meal-plan-dayhead meal-plan-weekhead">{t("mealPlan.week")}</div>

          {MEAL_ORDER.map((meal) => (
            <Fragment key={meal}>
              <div className="meal-plan-rowlabel">{t(mealLabelKey(meal))}</div>
              {MEAL_PLAN_DAYS.map((day) => {
                const cellItems = itemsForCell(items, day, meal);
                return (
                  <div key={`${meal}-${day}`} className="meal-plan-cell">
                    {cellItems.map((cellItem) => (
                      <div key={cellItem.id} className="meal-plan-chip">
                        <div className="meal-plan-chip-head">
                          <span className="meal-plan-chip-name">{cellItem.name}</span>
                          <span className="meal-plan-chip-kcal">
                            {t("mealPlan.kcal", { calories: cellItem.calories })}
                          </span>
                        </div>
                        {cellItem.amount && <span className="meal-plan-chip-amount">{cellItem.amount}</span>}
                        {cellItem.recipe_id && (
                          <span
                            className="meal-plan-chip-recipe"
                            title={t("mealPlan.recipeLinked", {
                              title: recipeTitles.get(cellItem.recipe_id) ?? t("mealPlan.recipeMissing"),
                            })}
                          >
                            🍽 {recipeTitles.get(cellItem.recipe_id) ?? t("mealPlan.recipeMissing")}
                          </span>
                        )}
                        <div className="meal-plan-chip-actions">
                          <button
                            className="btn btn-ghost btn-sm"
                            onClick={() => setEditor({ mode: "edit", item: cellItem })}
                          >
                            {t("common.edit")}
                          </button>
                          <button className="btn btn-ghost btn-sm" onClick={() => handleDelete(cellItem)}>
                            {t("common.delete")}
                          </button>
                        </div>
                      </div>
                    ))}
                    <button
                      className="meal-plan-add"
                      aria-label={t("mealPlan.addItemAria", {
                        meal: t(mealLabelKey(meal)),
                        day: t(dayLabelKey(day)),
                      })}
                      onClick={() => setEditor({ mode: "create", day, meal })}
                    >
                      {t("mealPlan.addItem")}
                    </button>
                  </div>
                );
              })}
              <div className="meal-plan-rowtotal">{t("mealPlan.kcal", { calories: mealTotals[meal] })}</div>
            </Fragment>
          ))}

          <div className="meal-plan-rowlabel meal-plan-total-label">{t("mealPlan.total")}</div>
          {MEAL_PLAN_DAYS.map((day) => (
            <div
              key={`total-${day}`}
              className="meal-plan-daytotal"
              aria-label={t("mealPlan.dayTotalAria", {
                day: t(dayLabelKey(day)),
                calories: dayTotals[day],
              })}
            >
              {t("mealPlan.kcal", { calories: dayTotals[day] })}
            </div>
          ))}
          <div className="meal-plan-rowtotal meal-plan-grandtotal">
            {t("mealPlan.kcal", { calories: weeklyTotal })}
          </div>
        </div>
      </div>

      {editor && <MealItemModal target={editor} onClose={() => setEditor(null)} />}
    </div>
  );
}

interface MealItemModalProps {
  target: EditorTarget;
  onClose: () => void;
}

/**
 * Create/edit dialog for one meal plan item. Creating fans out over the
 * selected days (one independent item each); editing keeps the item's day fixed
 * so a single row is replaced.
 */
function MealItemModal({ target, onClose }: MealItemModalProps) {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const existing = target.mode === "edit" ? target.item : null;
  const [days, setDays] = useState<MealPlanDay[]>(target.mode === "create" ? [target.day] : []);
  const [meal, setMeal] = useState<Meal>(target.mode === "create" ? target.meal : target.item.meal);
  const [name, setName] = useState(existing?.name ?? "");
  const [amount, setAmount] = useState(existing?.amount ?? "");
  const [calories, setCalories] = useState(String(existing?.calories ?? 0));
  const [recipeId, setRecipeId] = useState(existing?.recipe_id ?? "");
  const [error, setError] = useState<string | null>(null);

  const { data: recipes = [] } = useQuery({
    queryKey: ["recipes", "all"],
    queryFn: () => listRecipes(),
  });

  const saveMut = useMutation({
    mutationFn: async () => {
      const payload = {
        meal,
        name: name.trim(),
        amount: amount.trim(),
        calories: Number(calories || 0),
        recipe_id: recipeId || null,
      };
      if (existing) {
        const updated = await updateMealPlanItem(existing.id, { day: existing.day, ...payload });
        return [updated];
      }
      return createMealPlanItem({ days, ...payload });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MEAL_PLAN_KEY });
      onClose();
    },
    onError: (err: Error) => setError(err.message),
  });

  function toggleDay(day: MealPlanDay) {
    setDays((current) =>
      current.includes(day) ? current.filter((d) => d !== day) : [...current, day]
    );
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!existing && days.length === 0) {
      setError(t("mealPlan.validationDays"));
      return;
    }
    if (!name.trim()) {
      setError(t("mealPlan.validationName"));
      return;
    }
    const parsed = Number(calories || 0);
    if (!Number.isFinite(parsed) || parsed < 0) {
      setError(t("mealPlan.validationCalories"));
      return;
    }
    setError(null);
    saveMut.mutate();
  }

  return (
    <Modal
      onClose={onClose}
      maxWidth={520}
      ariaLabel={existing ? t("mealPlan.editTitle") : t("mealPlan.newTitle")}
    >
      <form onSubmit={handleSubmit}>
        <h3 className="modal-title">{existing ? t("mealPlan.editTitle") : t("mealPlan.newTitle")}</h3>

        <label className="meal-plan-field">
          <span>{t("mealPlan.meal")}</span>
          <select className="select" value={meal} onChange={(e) => setMeal(e.target.value as Meal)}>
            {MEAL_ORDER.map((option) => (
              <option key={option} value={option}>
                {t(mealLabelKey(option))}
              </option>
            ))}
          </select>
        </label>

        {existing ? (
          <label className="meal-plan-field">
            <span>{t("mealPlan.days")}</span>
            <span className="text-sm text-secondary">{t(dayLabelKey(existing.day))}</span>
          </label>
        ) : (
          <div className="meal-plan-field">
            <span>{t("mealPlan.days")}</span>
            <div className="meal-plan-daypicker">
              {MEAL_PLAN_DAYS.map((day) => (
                <button
                  key={day}
                  type="button"
                  className={`meal-plan-daybtn ${days.includes(day) ? "active" : ""}`}
                  aria-pressed={days.includes(day)}
                  onClick={() => toggleDay(day)}
                >
                  {t(dayLabelKey(day))}
                </button>
              ))}
            </div>
            <span className="text-sm text-secondary">{t("mealPlan.daysHint")}</span>
          </div>
        )}

        <label className="meal-plan-field">
          <span>{t("mealPlan.name")}</span>
          <input
            className="input"
            value={name}
            placeholder={t("mealPlan.namePlaceholder")}
            onChange={(e) => setName(e.target.value)}
          />
        </label>

        <label className="meal-plan-field">
          <span>{t("mealPlan.amount")}</span>
          <input
            className="input"
            value={amount}
            placeholder={t("mealPlan.amountPlaceholder")}
            onChange={(e) => setAmount(e.target.value)}
          />
        </label>

        <label className="meal-plan-field">
          <span>{t("mealPlan.calories")}</span>
          <input
            className="input"
            type="number"
            min={0}
            value={calories}
            onChange={(e) => setCalories(e.target.value)}
          />
        </label>

        <label className="meal-plan-field">
          <span>{t("mealPlan.recipe")}</span>
          <select className="select" value={recipeId ?? ""} onChange={(e) => setRecipeId(e.target.value)}>
            <option value="">{t("mealPlan.noRecipe")}</option>
            {recipes.map((recipe) => (
              <option key={recipe.id} value={recipe.id}>
                {recipe.title}
              </option>
            ))}
          </select>
        </label>

        {error && <p className="error-text text-sm">{error}</p>}

        <div className="modal-actions">
          <button type="button" className="btn" onClick={onClose}>
            {t("common.cancel")}
          </button>
          <button type="submit" className="btn btn-primary" disabled={saveMut.isPending}>
            {saveMut.isPending ? t("common.saving") : existing ? t("common.save") : t("mealPlan.add")}
          </button>
        </div>
      </form>
    </Modal>
  );
}
