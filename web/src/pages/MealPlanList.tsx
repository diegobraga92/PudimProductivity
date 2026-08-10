import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { listMealPlans, deleteMealPlan } from "../api/mealplan";
import { listRecipes } from "../api/recipes";

export default function MealPlanList({ onOpen }: { onOpen: (planId: string | null) => void }) {
  const queryClient = useQueryClient();
  const { data: plans = [], isLoading } = useQuery({ queryKey: ["mealplans"], queryFn: listMealPlans });
  const { data: recipes = [] } = useQuery({ queryKey: ["recipes", ""], queryFn: () => listRecipes({}) });

  const del = useMutation({
    mutationFn: (id: string) => deleteMealPlan(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["mealplans"] }),
  });

  return (
    <div className="animate-fade-in">
      <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-md)" }}>
        <h2 className="page-heading" style={{ marginBottom: 0 }}>🗓 Meal Plans</h2>
        <button className="btn btn-primary" onClick={() => onOpen(null)}>+ New plan</button>
      </div>

      {recipes.length === 0 && (
        <p style={{ color: "var(--color-warning)", marginBottom: "var(--space-md)" }}>
          Tip: create some recipes first so slots can reference them.
        </p>
      )}

      {isLoading && <p style={{ color: "var(--color-text-secondary)" }}>Loading plans…</p>}
      {plans.length === 0 && !isLoading && (
        <div className="empty-state">
          <div className="empty-state-icon">🗓</div>
          <p className="empty-state-text">No meal plans yet. Plan your week!</p>
        </div>
      )}

      <div style={{ display: "flex", flexDirection: "column", gap: "var(--space-sm)" }}>
        {plans.map((p) => (
          <div key={p.id} className="card" style={{ cursor: "pointer" }} onClick={() => onOpen(p.id)}>
            <div className="flex-center" style={{ justifyContent: "space-between" }}>
              <span className="card-title">{p.name || "Untitled plan"}</span>
              <span className={`badge ${p.is_published ? "badge-done" : "badge-todo"}`}>
                {p.is_published ? "Published" : "Draft"}
              </span>
            </div>
            <p style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
              {p.start_date} → {p.end_date} · {p.slots?.length ?? 0} slots
            </p>
            <button className="btn btn-danger btn-sm" onClick={(e) => { e.stopPropagation(); del.mutate(p.id); }}>Delete</button>
          </div>
        ))}
      </div>
    </div>
  );
}
