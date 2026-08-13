import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  createLibraryItem,
  deleteLibraryItem,
  listLibraryItems,
  updateLibraryItem,
  type CreateLibraryItemRequest,
  type LibraryItem,
  type MediaType,
} from "../api/library";
import { LibraryCsvImport } from "../components/LibraryCsvImport";

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

  const saving = create.isPending || update.isPending;
  const saveError = create.error ?? update.error ?? toggleDone.error ?? del.error;

  function openAdd() {
    setEditing(null);
    setForm(EMPTY_FORM);
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
    });
    setFormOpen(true);
  }

  function closeForm() {
    setFormOpen(false);
    setEditing(null);
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
            <textarea
              className="input"
              style={{ gridColumn: "1 / -1", minHeight: 56 }}
              placeholder="Notes (optional)"
              value={form.notes}
              onChange={(e) => setForm({ ...form, notes: e.target.value })}
            />
          </div>
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
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(260px, 1fr))", gap: "var(--space-md)" }}>
        {items.map((item) => (
          <div key={item.id} className="card" style={{ padding: "var(--space-md)" }}>
            <div className="flex-center" style={{ justifyContent: "space-between", gap: "var(--space-sm)" }}>
              <div style={{ minWidth: 0 }}>
                <p className="card-title" style={{ margin: 0, fontSize: "var(--font-size-sm)" }}>
                  {item.name}
                </p>
                <p className="text-sm text-secondary" style={{ margin: "0.2rem 0 0" }}>
                  {MEDIA_ICONS[item.media_type]} {MEDIA_LABELS[item.media_type] ?? item.media_type}
                  {item.release_year != null && <> · {item.release_year}</>}
                </p>
              </div>
              <label className="checkbox-wrapper" title={item.done ? "Mark as not done" : "Mark as done"}>
                <input
                  type="checkbox"
                  checked={item.done}
                  onChange={(e) => toggleDone.mutate({ id: item.id, done: e.target.checked })}
                />
                <span className="checkbox-custom" aria-hidden="true">
                  <svg viewBox="0 0 24 24" width="14" height="14">
                    <path fill="none" stroke="currentColor" strokeWidth="3" d="M20 6L9 17l-5-5" />
                  </svg>
                </span>
              </label>
            </div>
            {item.notes && (
              <p className="text-sm text-secondary" style={{ margin: "0.35rem 0 0" }} title={item.notes}>
                📝 {item.notes.length > 90 ? `${item.notes.slice(0, 90)}…` : item.notes}
              </p>
            )}
            <div className="flex-center" style={{ gap: "var(--space-sm)", marginTop: "0.5rem" }}>
              <button className="btn btn-sm" onClick={() => openEdit(item)}>Edit</button>
              <button className="btn btn-danger btn-sm" onClick={() => del.mutate(item.id)}>
                Delete
              </button>
              {item.done && <span className="badge badge-done">Done</span>}
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

