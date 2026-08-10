import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import config from "../config";
import {
  createMealPlan,
  generateShoppingList,
  getMealPlan,
  getShoppingList,
  publishMealPlan,
  toggleShoppingItem,
  type ShoppingItem,
} from "../api/mealplan";
import { listRecipes } from "../api/recipes";

function thisMonday(): string {
  const d = new Date();
  const day = (d.getDay() + 6) % 7; // 0 = Monday
  d.setDate(d.getDate() - day);
  return d.toISOString().slice(0, 10);
}

function addDays(date: string, days: number): string {
  const d = new Date(date + "T00:00:00Z");
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

export default function MealPlanDetail({ planId, onBack }: { planId: string | null; onBack: () => void }) {
  const queryClient = useQueryClient();
  const isNew = planId === null;
  const [name, setName] = useState("");
  const [startDate, setStartDate] = useState(thisMonday());
  const [endDate, setEndDate] = useState(addDays(thisMonday(), 6));

  const { data: plan } = useQuery({
    queryKey: ["mealplan", planId],
    queryFn: () => getMealPlan(planId!),
    enabled: !isNew,
  });
  const { data: recipes = [] } = useQuery({ queryKey: ["recipes", "all"], queryFn: () => listRecipes({}) });
  const { data: shopping = [] } = useQuery({
    queryKey: ["shopping", planId],
    queryFn: () => getShoppingList(planId!),
    enabled: !isNew,
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["mealplans"] });
  const create = useMutation({
    mutationFn: () => createMealPlan({ name, start_date: startDate, end_date: endDate, slots: [] }),
    onSuccess: () => onBack(),
  });
  const generate = useMutation({
    mutationFn: () => generateShoppingList(planId!),
    onSuccess: (items) => queryClient.setQueryData(["shopping", planId], items),
  });
  const toggle = useMutation({
    mutationFn: (itemId: string) => toggleShoppingItem(planId!, itemId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["shopping", planId] }),
  });
  const publish = useMutation({ mutationFn: () => publishMealPlan(planId!), onSuccess: invalidate });

  if (isNew) {
    return (
      <div className="animate-fade-in">
        <button className="btn btn-ghost btn-sm" onClick={onBack} style={{ marginBottom: "var(--space-md)" }}>← Back</button>
        <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700, marginBottom: "var(--space-md)" }}>🗓 New meal plan</h2>
        <div className="card" style={{ maxWidth: 480, padding: "var(--space-lg)" }}>
          <input className="input" style={{ width: "100%", marginBottom: "var(--space-md)" }} placeholder="Plan name" value={name} onChange={(e) => setName(e.target.value)} />
          <div className="flex-center" style={{ gap: "var(--space-md)", marginBottom: "var(--space-md)" }}>
            <label className="form-label-xs" style={{ display: "flex", alignItems: "center", gap: "0.3rem" }}>
              From <input className="input" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
            </label>
            <label className="form-label-xs" style={{ display: "flex", alignItems: "center", gap: "0.3rem" }}>
              To <input className="input" type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
            </label>
          </div>
          <button className="btn btn-primary" disabled={!startDate || !endDate} onClick={() => create.mutate()}>
            {create.isPending ? "Creating…" : "Create plan"}
          </button>
        </div>
      </div>
    );
  }

  const days: string[] = [];
  if (plan) {
    let d = plan.start_date;
    while (d <= plan.end_date) {
      days.push(d);
      d = addDays(d, 1);
    }
  }
  const slotFor = (date: string, meal: string) => plan?.slots?.find((s) => s.date === date && s.meal_type === meal);
  const recipeTitle = (id: string | null | undefined) => recipes.find((r) => r.id === id)?.title ?? "—";

  return (
    <div className="animate-fade-in">
      <button className="btn btn-ghost btn-sm" onClick={onBack} style={{ marginBottom: "var(--space-md)" }}>← Back</button>
      <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-md)" }}>
        <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700 }}>{plan?.name || "Meal plan"}</h2>
        <div className="flex-center" style={{ gap: "0.5rem" }}>
          {/* Phase 9b: printable PDF download */}
          {plan && (
            <a
              className="btn btn-sm"
              href={`${config.apiBaseUrl}/mealplans/${plan.id}/pdf`}
              download
              title="Download the weekly grid + shopping list as a PDF"
            >
              📄 PDF
            </a>
          )}
          <button className="btn btn-sm" disabled={publish.isPending} onClick={() => publish.mutate()}>
            {plan?.is_published ? "✓ Published" : "Publish"}
          </button>
        </div>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))", gap: "var(--space-sm)", marginBottom: "var(--space-lg)" }}>
        {days.map((day) => (
          <div key={day} className="card" style={{ padding: "var(--space-sm)" }}>
            <p style={{ fontWeight: 600, fontSize: "var(--font-size-sm)", marginBottom: "0.3rem" }}>{day}</p>
            {["breakfast", "lunch", "dinner", "snack"].map((meal) => {
              const slot = slotFor(day, meal);
              return (
                <div key={meal} style={{ marginBottom: "0.25rem" }}>
                  <span style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>{meal}: </span>
                  <span style={{ fontSize: "var(--font-size-xs)" }}>{slot ? recipeTitle(slot.recipe_id) : "—"}</span>
                </div>
              );
            })}
          </div>
        ))}
      </div>

      <div className="card" style={{ padding: "var(--space-md)", marginBottom: "var(--space-md)" }}>
        <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-sm)" }}>
          <h3 style={{ fontSize: "var(--font-size-lg)", fontWeight: 600 }}>🛒 Shopping list</h3>
          <button className="btn btn-sm" disabled={generate.isPending} onClick={() => generate.mutate()}>
            {generate.isPending ? "Generating…" : "Regenerate"}
          </button>
        </div>
        {shopping.length === 0 && (
          <p style={{ color: "var(--color-text-secondary)", fontSize: "var(--font-size-sm)" }}>No list yet — generate from the slotted recipes.</p>
        )}
        {shopping.map((item: ShoppingItem) => (
          <label key={item.id} className="flex-center" style={{ gap: "0.5rem", cursor: "pointer", padding: "0.25rem 0" }}>
            <input type="checkbox" checked={item.is_checked} onChange={() => toggle.mutate(item.id)} />
            <span style={{ textDecoration: item.is_checked ? "line-through" : "none", color: item.is_checked ? "var(--color-text-secondary)" : "inherit" }}>
              {item.quantity_agg ? `${item.quantity_agg} ${item.unit}`.trim() + " " : ""}{item.ingredient_name}
            </span>
          </label>
        ))}
      </div>
    </div>
  );
}

