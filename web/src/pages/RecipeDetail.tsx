import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
import {
  createRecipe,
  generateRecipeUploadURL,
  getRecipe,
  resolveMediaUrl,
  updateRecipe,
  uploadToPresignedUrl,
} from "../api/recipes";
import { useI18n } from "../i18n";

type IngredientRow = { name: string; quantity: string; unit: string };
type StepRow = { instruction: string };

export default function RecipeDetail({ recipeId, onBack }: { recipeId: string; onBack: () => void }) {
  const isNew = recipeId === "__new__";
  const queryClient = useQueryClient();
  const { t } = useI18n();

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
  // Image + source link (stored on the recipe via image_url / source_url).
  const [imageUrl, setImageUrl] = useState("");
  const [sourceUrl, setSourceUrl] = useState("");
  const [pendingImage, setPendingImage] = useState<File | null>(null);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Preview the chosen file (revoked on change/unmount to avoid leaks).
  const previewUrl = useMemo(
    () => (pendingImage ? URL.createObjectURL(pendingImage) : null),
    [pendingImage]
  );
  useEffect(() => {
    return () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl);
    };
  }, [previewUrl]);

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
    setImageUrl(recipe.image_url ?? "");
    setSourceUrl(recipe.source_url ?? "");
    setHydrated(true);
  }

  const saveMut = useMutation({
    mutationFn: async () => {
      setSaveError(null);
      const baseBody = {
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

      // New recipes need an id before an image can be uploaded, so create
      // first (without media), then upload, then save the final record.
      let id = recipeId;
      if (isNew) {
        const created = await createRecipe({ ...baseBody, image_url: null, source_url: null });
        id = created.id;
      }

      let image = imageUrl.trim() || null;
      if (pendingImage) {
        const upload = await generateRecipeUploadURL(id, {
          content_type: pendingImage.type || "image/jpeg",
          filename: pendingImage.name || "image.jpg",
        });
        await uploadToPresignedUrl(upload.url, pendingImage);
        image = upload.key;
      }

      return updateRecipe(id, {
        ...baseBody,
        image_url: image,
        source_url: sourceUrl.trim() || null,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["recipes"] });
      onBack();
    },
    onError: (err: Error) => setSaveError(err.message),
  });

  // Image to show in the preview (a newly chosen file wins over the stored URL).
  const currentImage = previewUrl ?? resolveMediaUrl(imageUrl);

  return (
    <div className="animate-fade-in">
      <button className="btn btn-ghost btn-sm" onClick={onBack} style={{ marginBottom: "var(--space-md)" }}>
        ← {t("common.back")}
      </button>
      <h2 className="page-heading" style={{ marginBottom: "var(--space-md)" }}>
        {isNew ? t("recipes.newTitle") : t("recipes.editTitle")}
      </h2>

      <div className="card" style={{ maxWidth: 640, padding: "var(--space-lg)" }}>
        <div className="flex-center" style={{ gap: "var(--space-md)", marginBottom: "var(--space-md)" }}>
          <input className="input" style={{ flex: 2 }} placeholder={t("common.title")} value={title} onChange={(e) => setTitle(e.target.value)} />
          <select className="select" value={difficulty} onChange={(e) => setDifficulty(e.target.value as "easy" | "medium" | "hard")}>
            <option value="easy">{t("recipes.easy")}</option>
            <option value="medium">{t("recipes.medium")}</option>
            <option value="hard">{t("recipes.hard")}</option>
          </select>
        </div>
        <input
          className="input" style={{ width: "100%", marginBottom: "var(--space-md)" }}
          placeholder={t("recipes.description")} value={description} onChange={(e) => setDescription(e.target.value)}
        />
        <div className="flex-center" style={{ gap: "var(--space-md)", marginBottom: "var(--space-md)" }}>
          <label className="form-label-xs" style={{ display: "flex", alignItems: "center", gap: "0.3rem" }}>
            {t("recipes.prep")} <input className="input" type="number" style={{ width: 70 }} value={prep} onChange={(e) => setPrep(Number(e.target.value))} />
          </label>
          <label className="form-label-xs" style={{ display: "flex", alignItems: "center", gap: "0.3rem" }}>
            {t("recipes.cook")} <input className="input" type="number" style={{ width: 70 }} value={cook} onChange={(e) => setCook(Number(e.target.value))} />
          </label>
          <label className="form-label-xs" style={{ display: "flex", alignItems: "center", gap: "0.3rem" }}>
            {t("recipes.servings")} <input className="input" type="number" style={{ width: 70 }} value={servings} onChange={(e) => setServings(Number(e.target.value))} />
          </label>
        </div>
        <input
          className="input" style={{ width: "100%", marginBottom: "var(--space-md)" }}
          placeholder={t("recipes.tags")} value={tags} onChange={(e) => setTags(e.target.value)}
        />
        <input
          className="input" style={{ width: "100%", marginBottom: "var(--space-md)" }}
          placeholder={t("recipes.sourceLink")} value={sourceUrl} onChange={(e) => setSourceUrl(e.target.value)}
        />

        {/* Image */}
        <label className="form-label">{t("recipes.image")}</label>
        <div className="flex-center" style={{ gap: "var(--space-md)", marginBottom: "var(--space-sm)" }}>
          {currentImage ? (
            <img
              src={currentImage}
              alt={t("recipes.previewAlt")}
              style={{ width: 96, height: 72, objectFit: "cover", borderRadius: "8px", flexShrink: 0 }}
            />
          ) : (
            <div
              className="recipe-thumb-placeholder"
              style={{ width: 96, height: 72, display: "flex", alignItems: "center", justifyContent: "center", flexShrink: 0 }}
            >
              🍽️
            </div>
          )}
          <input
            type="file"
            accept="image/*"
            onChange={(e) => {
              const file = e.target.files?.[0];
              e.target.value = "";
              if (!file) return;
              if (file.size > 10 * 1024 * 1024) {
                setUploadError(t("recipes.imageTooLarge"));
                return;
              }
              setUploadError(null);
              setPendingImage(file);
            }}
          />
        </div>
        <input
          className="input" style={{ width: "100%", marginBottom: "var(--space-sm)" }}
          placeholder={t("recipes.imageUrl")} value={imageUrl} onChange={(e) => setImageUrl(e.target.value)}
        />
        {pendingImage && (
          <p className="text-sm text-secondary" style={{ marginBottom: "var(--space-sm)" }}>
            {t("recipes.pendingImage", { name: pendingImage.name })}
          </p>
        )}
        {uploadError && (
          <p className="text-sm" style={{ color: "var(--color-danger)", marginBottom: "var(--space-sm)" }}>
            {uploadError}
          </p>
        )}

        <label className="form-label" style={{ marginTop: "var(--space-md)" }}>{t("recipes.ingredients")}</label>
        {ingredients.map((ing, i) => (
          <div key={i} className="flex-center" style={{ gap: "0.4rem", marginBottom: "0.35rem" }}>
            <input className="input" placeholder={t("recipes.ingName")} value={ing.name} onChange={(e) => updateRow("ingredients", i, "name", e.target.value)} />
            <input className="input" style={{ width: 80 }} placeholder={t("recipes.ingQty")} value={ing.quantity} onChange={(e) => updateRow("ingredients", i, "quantity", e.target.value)} />
            <input className="input" style={{ width: 90 }} placeholder={t("recipes.ingUnit")} value={ing.unit} onChange={(e) => updateRow("ingredients", i, "unit", e.target.value)} />
            <button className="btn btn-danger btn-sm" onClick={() => removeRow("ingredients", i)}>✕</button>
          </div>
        ))}
        <button className="btn btn-sm" onClick={() => setIngredients([...ingredients, { name: "", quantity: "", unit: "" }])}>
          {t("recipes.addIngredient")}
        </button>

        <label className="form-label" style={{ marginTop: "var(--space-md)" }}>{t("recipes.steps")}</label>
        {steps.map((s, i) => (
          <div key={i} className="flex-center" style={{ gap: "0.4rem", marginBottom: "0.35rem" }}>
            <input className="input" style={{ flex: 1 }} placeholder={t("recipes.stepPlaceholder", { number: i + 1 })} value={s.instruction} onChange={(e) => updateRow("steps", i, "instruction", e.target.value)} />
            <button className="btn btn-danger btn-sm" onClick={() => removeRow("steps", i)}>✕</button>
          </div>
        ))}
        <button className="btn btn-sm" onClick={() => setSteps([...steps, { instruction: "" }])}>{t("recipes.addStep")}</button>

        <div style={{ marginTop: "var(--space-lg)" }}>
          {saveError && (
            <p className="text-sm" style={{ color: "var(--color-danger)", marginBottom: "var(--space-sm)" }}>
              {saveError}
            </p>
          )}
          <button className="btn btn-primary" disabled={saveMut.isPending} onClick={() => saveMut.mutate()}>
            {saveMut.isPending ? t("common.saving") : isNew ? t("recipes.create") : t("recipes.saveChanges")}
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

