import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { createRecipe, getRecipe, updateRecipe } from "../api/recipes";

type IngredientRow = { name: string; quantity: string; unit: string };
type StepRow = { instruction: string };

export default function RecipeDetail({ recipeId, onBack }: { recipeId: string; onBack: () => void }) {
  const isNew = recipeId === "__new__";
  const queryClient = useQueryClient();

  const { data: recipe } = useQuery({
    queryKey: ["recipe", recipeId],
    queryFn: () => getRecipe(recipeId),
    enabled: !isNew,
  });

  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [difficulty, setDifficulty] = useState<"easy" | "medium" | "hard">("easy");
  const [prep, setPrep] = useState(10);
  const [cook, setCook] = useState(10);
  const [servings, setServings] = useState(2);
  const [tags, setTags] = useState("");
  const [ingredients, setIngredients] = useState<IngredientRow[]>([{ name: "", quantity: "", unit: "" }]);
  const [steps, setSteps] = useState<StepRow[]>([{ instruction: "" }]);

  // Hydrate the form once the fetched recipe arrives (edit mode).
  const [hydrated, setHydrated] = useState(false);
  if (recipe && !hydrated && !isNew) {
    setTitle(recipe.title);
    setDescription(recipe.description ?? "");
    setDifficulty(recipe.difficulty);
    setPrep(recipe.prep_time_minutes);
    setCook(recipe.cook_time_minutes);
    setServings(recipe.servings);
    setTags((recipe.tags ?? []).join(", "));
    setIngredients(
      recipe.ingredients?.length
        ? recipe.ingredients.map((i) => ({ name: i.name, quantity: i.quantity ?? "", unit: i.unit ?? "" }))
        : []
    );
    setSteps(recipe.steps?.length ? recipe.steps.map((s) => ({ instruction: s.instruction })) : []);
    setHydrated(true);
  }

  const saveMut = useMutation({
    mutationFn: () => {
      const body = {
        title,
        description,
        difficulty,
        prep_time_minutes: prep,
        cook_time_minutes: cook,
        servings,
        tags: tags.split(",").map((t) => t.trim()).filter(Boolean),
        ingredients: ingredients.filter((i) => i.name.trim()),
        steps: steps.filter((s) => s.instruction.trim()),
      };
      return isNew ? createRecipe(body) : updateRecipe(recipeId, body);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recipes"] });
      onBack();
    },
  });

  return (
    <div className="animate-fade-in">
      <button className="btn btn-ghost btn-sm" onClick={onBack} style={{ marginBottom: "var(--space-md)" }}>
        ← Back
      </button>
      <h2 className="page-heading" style={{ marginBottom: "var(--space-md)" }}>
        {isNew ? "🍳 New recipe" : "✏️ Edit recipe"}
      </h2>

      <div className="card" style={{ maxWidth: 640, padding: "var(--space-lg)" }}>
        <div className="flex-center" style={{ gap: "var(--space-md)", marginBottom: "var(--space-md)" }}>
          <input className="input" style={{ flex: 2 }} placeholder="Title" value={title} onChange={(e) => setTitle(e.target.value)} />
          <select className="select" value={difficulty} onChange={(e) => setDifficulty(e.target.value as "easy" | "medium" | "hard")}>
            <option value="easy">Easy</option>
            <option value="medium">Medium</option>
            <option value="hard">Hard</option>
          </select>
        </div>
        <input
          className="input" style={{ width: "100%", marginBottom: "var(--space-md)" }}
          placeholder="Description (optional)" value={description} onChange={(e) => setDescription(e.target.value)}
        />
        <div className="flex-center" style={{ gap: "var(--space-md)", marginBottom: "var(--space-md)" }}>
          <label className="form-label-xs" style={{ display: "flex", alignItems: "center", gap: "0.3rem" }}>
            Prep (min) <input className="input" type="number" style={{ width: 70 }} value={prep} onChange={(e) => setPrep(Number(e.target.value))} />
          </label>
          <label className="form-label-xs" style={{ display: "flex", alignItems: "center", gap: "0.3rem" }}>
            Cook (min) <input className="input" type="number" style={{ width: 70 }} value={cook} onChange={(e) => setCook(Number(e.target.value))} />
          </label>
          <label className="form-label-xs" style={{ display: "flex", alignItems: "center", gap: "0.3rem" }}>
            Servings <input className="input" type="number" style={{ width: 70 }} value={servings} onChange={(e) => setServings(Number(e.target.value))} />
          </label>
        </div>
        <input
          className="input" style={{ width: "100%", marginBottom: "var(--space-md)" }}
          placeholder="Tags (comma separated)" value={tags} onChange={(e) => setTags(e.target.value)}
        />

        <label className="form-label">Ingredients</label>
        {ingredients.map((ing, i) => (
          <div key={i} className="flex-center" style={{ gap: "0.4rem", marginBottom: "0.35rem" }}>
            <input className="input" placeholder="Name" value={ing.name} onChange={(e) => updateRow("ingredients", i, "name", e.target.value)} />
            <input className="input" style={{ width: 80 }} placeholder="Qty" value={ing.quantity} onChange={(e) => updateRow("ingredients", i, "quantity", e.target.value)} />
            <input className="input" style={{ width: 90 }} placeholder="Unit" value={ing.unit} onChange={(e) => updateRow("ingredients", i, "unit", e.target.value)} />
            <button className="btn btn-danger btn-sm" onClick={() => removeRow("ingredients", i)}>✕</button>
          </div>
        ))}
        <button className="btn btn-sm" onClick={() => setIngredients([...ingredients, { name: "", quantity: "", unit: "" }])}>
          + Add ingredient
        </button>

        <label className="form-label" style={{ marginTop: "var(--space-md)" }}>Steps</label>
        {steps.map((s, i) => (
          <div key={i} className="flex-center" style={{ gap: "0.4rem", marginBottom: "0.35rem" }}>
            <input className="input" style={{ flex: 1 }} placeholder={`Step ${i + 1}`} value={s.instruction} onChange={(e) => updateRow("steps", i, "instruction", e.target.value)} />
            <button className="btn btn-danger btn-sm" onClick={() => removeRow("steps", i)}>✕</button>
          </div>
        ))}
        <button className="btn btn-sm" onClick={() => setSteps([...steps, { instruction: "" }])}>+ Add step</button>

        <div style={{ marginTop: "var(--space-lg)" }}>
          <button className="btn btn-primary" disabled={saveMut.isPending} onClick={() => saveMut.mutate()}>
            {saveMut.isPending ? "Saving…" : isNew ? "Create recipe" : "Save changes"}
          </button>
        </div>
      </div>
    </div>
  );

  function updateRow(kind: "ingredients" | "steps", index: number, field: string, value: string) {
    if (kind === "ingredients") {
      setIngredients(ingredients.map((r, i) => (i === index ? { ...r, [field]: value } : r)));
    } else {
      setSteps(steps.map((r, i) => (i === index ? { ...r, [field]: value } : r)));
    }
  }

  function removeRow(kind: "ingredients" | "steps", index: number) {
    if (kind === "ingredients") {
      setIngredients(ingredients.filter((_, i) => i !== index));
    } else {
      setSteps(steps.filter((_, i) => i !== index));
    }
  }
}

