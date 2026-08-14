import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  createLibraryItem,
  deleteLibraryItem,
  listLibraryItems,
  searchLibraryScores,
  updateLibraryItem,
  type CreateLibraryItemRequest,
  type LibraryItem,
  type MediaType,
  type ScoreCandidate,
} from "../api/library";
import { LibraryCsvImport } from "../components/LibraryCsvImport";
import { useConfirm } from "../components/useConfirm";

const MEDIA_TYPES: MediaType[] = ["movie", "series", "book", "game"];

const MEDIA_LABELS: Record<MediaType, string> = {
  movie: "Movie",
  series: "Series",
  book: "Book",
  game: "Game",
};

const MEDIA_ICONS: Record<MediaType, string> = {
  movie: "🎬",
  series: "📺",
  book: "📚",
  game: "🎮",
};

const EMPTY_FORM: CreateLibraryItemRequest = {
  name: "",
  media_type: "book",
  release_year: null,
  done: false,
  notes: "",
  subtype: "",
  score: null,
  score_source: "",
};

/**
 * Library: a simple tracker for movies, series, books and games. Shows the
 * name, media type, release year and a done flag, with optional notes and CSV
 * import. Replaces the old Books page.
 */
export default function Library() {
  const queryClient = useQueryClient();
  const [typeFilter, setTypeFilter] = useState("");
  const [doneFilter, setDoneFilter] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<LibraryItem | null>(null);
  const [form, setForm] = useState<CreateLibraryItemRequest>(EMPTY_FORM);
  const [importOpen, setImportOpen] = useState(false);
  const [scoreHits, setScoreHits] = useState<ScoreCandidate[] | null>(null);
  const [scoreSearching, setScoreSearching] = useState(false);
  const [scoreError, setScoreError] = useState<string | null>(null);
  // Ids selected for bulk actions (e.g. delete).
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const confirm = useConfirm();

  const { data: items = [], isLoading } = useQuery({
    queryKey: ["library", typeFilter, doneFilter],
    queryFn: () =>
      listLibraryItems(
        (typeFilter || undefined) as MediaType | undefined,
        doneFilter === "" ? undefined : doneFilter === "true",
      ),
  });

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["library"] });

  const create = useMutation({
    mutationFn: () => createLibraryItem(form),
    onSuccess: () => {
      invalidate();
      closeForm();
    },
  });

  const update = useMutation({
    mutationFn: () =>
      updateLibraryItem(editing!.id, {
        name: form.name,
        media_type: form.media_type,
        release_year: form.release_year,
        done: form.done,
        notes: form.notes,
        subtype: form.subtype,
        score: form.score,
        score_source: form.score_source,
      }),
    onSuccess: () => {
      invalidate();
      closeForm();
    },
  });

  const toggleDone = useMutation({
    mutationFn: ({ id, done }: { id: string; done: boolean }) =>
      updateLibraryItem(id, { done }),
    onSuccess: invalidate,
  });

  const del = useMutation({
    mutationFn: (id: string) => deleteLibraryItem(id),
    onSuccess: invalidate,
  });

  const bulkDelete = useMutation({
    mutationFn: (ids: string[]) => Promise.all(ids.map((id) => deleteLibraryItem(id))),
    onSuccess: () => {
      invalidate();
      setSelected(new Set());
    },
  });

  const saving = create.isPending || update.isPending;
  const saveError = create.error ?? update.error ?? toggleDone.error ?? del.error ?? bulkDelete.error;

  function openAdd() {
    setEditing(null);
    setForm(EMPTY_FORM);
    setScoreHits(null);
    setScoreError(null);
    setFormOpen(true);
  }

  function openEdit(item: LibraryItem) {
    setEditing(item);
    setForm({
      name: item.name,
      media_type: item.media_type,
      release_year: item.release_year,
      done: item.done,
      notes: item.notes,
      subtype: item.subtype ?? "",
      score: item.score ?? null,
      score_source: item.score_source ?? "",
    });
    setScoreHits(null);
    setScoreError(null);
    setFormOpen(true);
  }

  function closeForm() {
    setFormOpen(false);
    setEditing(null);
    setScoreHits(null);
    setScoreError(null);
  }

  /** Queries the configured rating provider for the form's current title. */
  async function lookUpScore() {
    const title = form.name.trim();
    if (!title) return;
    setScoreSearching(true);
    setScoreError(null);
    setScoreHits(null);
    try {
      const hits = await searchLibraryScores(title, form.media_type, form.release_year ?? undefined);
      setScoreHits(hits);
      if (hits.length === 0) setScoreError("No ratings found for this title.");
    } catch (err) {
      setScoreError(err instanceof Error ? err.message : String(err));
    } finally {
      setScoreSearching(false);
    }
  }

  /** Applies a confirmed candidate's score + source to the form. */
  function applyCandidate(candidate: ScoreCandidate) {
    setForm({
      ...form,
      score: candidate.score,
      score_source: candidate.score_source,
      name: candidate.title.trim() ? candidate.title.trim() : form.name,
      release_year:
        candidate.year != null && candidate.year > 0 ? candidate.year : form.release_year,
    });
    setScoreHits(null);
  }

  function save() {
    if (editing) {
      update.mutate();
    } else {
      create.mutate();
    }
  }

  return (
    <div className="animate-fade-in">
      <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-md)" }}>
        <h2 className="page-heading" style={{ marginBottom: 0 }}>🎬 Library</h2>
      </div>

      {/* Toolbar: add / import / filters */}
      <div className="flex-center" style={{ gap: "var(--space-sm)", marginBottom: "var(--space-md)", flexWrap: "wrap" }}>
        <button className="btn btn-primary" onClick={openAdd} disabled={formOpen}>
          {formOpen ? "Editing…" : "+ Add item"}
        </button>
        <button className="btn" onClick={() => setImportOpen(true)}>📥 Import CSV</button>
        <select
          className="select"
          value={typeFilter}
          onChange={(e) => setTypeFilter(e.target.value)}
          aria-label="Filter by type"
        >
          <option value="">All types</option>
          {MEDIA_TYPES.map((t) => (
            <option key={t} value={t}>
              {MEDIA_ICONS[t]} {MEDIA_LABELS[t]}
            </option>
          ))}
        </select>
        <select
          className="select"
          value={doneFilter}
          onChange={(e) => setDoneFilter(e.target.value)}
          aria-label="Filter by status"
        >
          <option value="">All</option>
          <option value="true">Done</option>
          <option value="false">Not done</option>
        </select>
      </div>

      {/* Add / edit form */}
      {formOpen && (
        <div className="card" style={{ maxWidth: 520, padding: "var(--space-md)", marginBottom: "var(--space-md)" }}>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))", gap: "0.4rem", marginBottom: "0.4rem" }}>
            <input
              className="input"
              style={{ gridColumn: "1 / -1" }}
              placeholder="Name (e.g. The Matrix)"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              autoFocus
            />
            <select
              className="select"
              value={form.media_type}
              onChange={(e) => setForm({ ...form, media_type: e.target.value as MediaType })}
            >
              {MEDIA_TYPES.map((t) => (
                <option key={t} value={t}>
                  {MEDIA_ICONS[t]} {MEDIA_LABELS[t]}
                </option>
              ))}
            </select>
            <input
              className="input"
              type="number"
              min={1800}
              max={2100}
              placeholder="Year (e.g. 1999)"
              value={form.release_year ?? ""}
              onChange={(e) =>
                setForm({
                  ...form,
                  release_year: e.target.value === "" ? null : Number(e.target.value),
                })
              }
            />
            <input
              className="input"
              placeholder={form.media_type === "game" ? "Console (e.g. PlayStation 5)" : "Genre (e.g. Sci-fi)"}
              value={form.subtype}
              onChange={(e) => setForm({ ...form, subtype: e.target.value })}
            />
            <textarea
              className="input"
              style={{ gridColumn: "1 / -1", minHeight: 56 }}
              placeholder="Notes (optional)"
              value={form.notes}
              onChange={(e) => setForm({ ...form, notes: e.target.value })}
            />
          </div>

          {/* Score: manual entry or looked up from a configured provider */}
          <div className="flex-center" style={{ gap: "0.4rem", marginBottom: "0.4rem", flexWrap: "wrap" }}>
            <input
              className="input"
              type="number"
              min={0}
              max={100}
              step="0.1"
              placeholder="Score (e.g. 8.7)"
              value={form.score ?? ""}
              onChange={(e) =>
                setForm({ ...form, score: e.target.value === "" ? null : Number(e.target.value) })
              }
              style={{ flex: 1, minWidth: 120 }}
            />
            <input
              className="input"
              placeholder="Source (e.g. imdb, metacritic)"
              value={form.score_source}
              onChange={(e) => setForm({ ...form, score_source: e.target.value })}
              style={{ flex: 1, minWidth: 140 }}
            />
            <button
              type="button"
              className="btn btn-sm"
              disabled={!form.name.trim() || scoreSearching}
              onClick={lookUpScore}
            >
              {scoreSearching ? "Searching…" : "⭐ Look up score"}
            </button>
          </div>

          {scoreError && (
            <p className="text-sm" style={{ color: "var(--color-danger)", margin: "0 0 0.4rem" }}>
              {scoreError}
            </p>
          )}

          {scoreHits && scoreHits.length > 0 && (
            <div style={{ marginBottom: "0.4rem" }}>
              <p className="text-sm" style={{ fontWeight: 600 }}>Pick a match:</p>
              {scoreHits.map((hit) => (
                <button
                  key={hit.external_id || hit.title}
                  type="button"
                  className="btn btn-sm"
                  style={{ margin: "0 0.25rem 0.25rem 0" }}
                  onClick={() => applyCandidate(hit)}
                >
                  {hit.title}
                  {hit.year != null && hit.year > 0 ? ` (${hit.year})` : ""} · ★ {hit.score}
                  {hit.score_source ? ` (${hit.score_source})` : ""}
                </button>
              ))}
            </div>
          )}

          <label className="flex-center" style={{ gap: "var(--space-sm)", marginBottom: "0.4rem" }}>
            <input
              type="checkbox"
              checked={form.done}
              onChange={(e) => setForm({ ...form, done: e.target.checked })}
            />
            <span className="text-sm">Done (consumed / read / watched / played)</span>
          </label>
          <div className="flex-center" style={{ gap: "var(--space-sm)" }}>
            <button
              className="btn btn-primary btn-sm"
              disabled={!form.name.trim() || saving}
              onClick={save}
            >
              {saving ? "Saving…" : editing ? "Save changes" : "Add"}
            </button>
            <button className="btn btn-ghost btn-sm" onClick={closeForm}>Cancel</button>
          </div>
        </div>
      )}

      {saveError && <p style={{ color: "var(--color-danger)" }}>{String(saveError)}</p>}
      {isLoading && <p style={{ color: "var(--color-text-secondary)" }}>Loading library…</p>}
      {items.length === 0 && !isLoading && (
        <div className="empty-state">
          <div className="empty-state-icon">🎬</div>
          <p className="empty-state-text">
            Your library is empty. Add an item above, or import a CSV.
          </p>
        </div>
      )}


      {/* Item list */}
      {/* Bulk actions */}
      {selected.size > 0 && (
        <div
          className="flex-center"
          style={{ gap: "var(--space-sm)", marginBottom: "var(--space-sm)", flexWrap: "wrap" }}
        >
          <span className="text-sm text-secondary">{selected.size} selected</span>
          <button
            className="btn btn-danger btn-sm"
            disabled={bulkDelete.isPending}
            onClick={async () => {
              const ok = await confirm({
                title: `Delete ${selected.size} item(s)?`,
                message: "This cannot be undone.",
                confirmLabel: "Delete",
                confirmVariant: "danger",
              });
              if (ok) bulkDelete.mutate([...selected]);
            }}
          >
            {bulkDelete.isPending ? "Deleting…" : `🗑 Delete selected (${selected.size})`}
          </button>
          <button className="btn btn-ghost btn-sm" onClick={() => setSelected(new Set())}>
            Clear
          </button>
        </div>
      )}

      <div
        className="library-list"
        style={{
          display: "flex",
          flexDirection: "column",
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-md)",
          overflowX: "auto",
        }}
      >
        {/* Header row */}
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "28px minmax(160px, 2fr) 90px 130px 70px 80px 170px",
            gap: "var(--space-sm)",
            alignItems: "center",
            padding: "0.5rem var(--space-md)",
            background: "var(--color-surface)",
            borderBottom: "1px solid var(--color-border)",
            fontWeight: 600,
            fontSize: "var(--font-size-sm)",
            minWidth: 640,
          }}
        >
          <label className="checkbox-wrapper" title="Select all">
            <input
              type="checkbox"
              checked={items.length > 0 && selected.size === items.length}
              onChange={(e) =>
                setSelected(e.target.checked ? new Set(items.map((i) => i.id)) : new Set())
              }
            />
            <span className="checkbox-custom" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="12" height="12">
                <path fill="none" stroke="currentColor" strokeWidth="3" d="M20 6L9 17l-5-5" />
              </svg>
            </span>
          </label>
          <span>Name</span>
          <span>Type</span>
          <span>Subtype</span>
          <span>Year</span>
          <span>Score</span>
          <span>Actions</span>
        </div>

        {items.map((item) => (
          <div
            key={item.id}
            style={{
              display: "grid",
              gridTemplateColumns: "28px minmax(160px, 2fr) 90px 130px 70px 80px 170px",
              gap: "var(--space-sm)",
              alignItems: "center",
              padding: "0.5rem var(--space-md)",
              borderBottom: "1px solid var(--color-border-light)",
              background: selected.has(item.id) ? "var(--color-primary-subtle)" : undefined,
              minWidth: 640,
            }}
          >
            <label className="checkbox-wrapper" title="Select for bulk actions">
              <input
                type="checkbox"
                checked={selected.has(item.id)}
                onChange={(e) => {
                  const next = new Set(selected);
                  if (e.target.checked) next.add(item.id);
                  else next.delete(item.id);
                  setSelected(next);
                }}
              />
              <span className="checkbox-custom" aria-hidden="true">
                <svg viewBox="0 0 24 24" width="12" height="12">
                  <path fill="none" stroke="currentColor" strokeWidth="3" d="M20 6L9 17l-5-5" />
                </svg>
              </span>
            </label>
            <div style={{ minWidth: 0 }}>
              <p className="card-title" style={{ margin: 0, fontSize: "var(--font-size-sm)" }}>
                {item.name}
              </p>
              {item.notes && (
                <p className="text-sm text-secondary" style={{ margin: "0.1rem 0 0" }} title={item.notes}>
                  📝 {item.notes.length > 60 ? `${item.notes.slice(0, 60)}…` : item.notes}
                </p>
              )}
            </div>
            <span className="text-sm">
              {MEDIA_ICONS[item.media_type]} {MEDIA_LABELS[item.media_type] ?? item.media_type}
            </span>
            <span className="text-sm" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
              {item.subtype || <span style={{ color: "var(--color-text-muted)" }}>—</span>}
            </span>
            <span className="text-sm">{item.release_year ?? "—"}</span>
            <span className="text-sm">{item.score != null ? `★ ${item.score}` : "—"}</span>
            <div className="flex-center" style={{ gap: "0.35rem" }}>
              <label className="checkbox-wrapper" title={item.done ? "Mark as not done" : "Mark as done"}>
                <input
                  type="checkbox"
                  checked={item.done}
                  onChange={(e) => toggleDone.mutate({ id: item.id, done: e.target.checked })}
                />
                <span className="checkbox-custom" aria-hidden="true">
                  <svg viewBox="0 0 24 24" width="12" height="12">
                    <path fill="none" stroke="currentColor" strokeWidth="3" d="M20 6L9 17l-5-5" />
                  </svg>
                </span>
              </label>
              <button className="btn btn-sm" onClick={() => openEdit(item)}>Edit</button>
              <button className="btn btn-danger btn-sm" onClick={() => del.mutate(item.id)}>
                Delete
              </button>
            </div>
          </div>
        ))}
      </div>

      {importOpen && (
        <LibraryCsvImport
          onClose={() => setImportOpen(false)}
          onImported={() => invalidate()}
        />
      )}
    </div>
  );
}

