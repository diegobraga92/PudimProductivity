import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import {
  createLibraryItem,
  deleteLibraryItem,
  listLibraryItems,
  listLibrarySubtypes,
  searchLibraryScores,
  updateLibraryItem,
  type CreateLibraryItemRequest,
  type LibraryItem,
  type MediaType,
  type ScoreCandidate,
} from "../api/library";
import { LibraryBulkScore } from "../components/LibraryBulkScore";
import { LibraryCsvImport } from "../components/LibraryCsvImport";
import Checkbox from "../components/Checkbox";
import { FilmIcon } from "../components/icons";
import Modal from "../components/Modal";
import { useConfirm } from "../components/useConfirm";
import { useFeatureFlag } from "../hooks/useFeatureFlag";
import { useI18n } from "../i18n";
import { normalizeScore } from "../utils/csv";
import { MEDIA_ICONS, MEDIA_LABEL_KEYS, MEDIA_TYPES } from "../utils/media";

const formatScore = (score: number): string => String(normalizeScore(score));

/** Shared column track list for the library table header and rows. */
const LIBRARY_GRID = "28px minmax(160px, 2fr) 90px 130px 70px 80px 170px";

type SortKey = "name" | "media_type" | "subtype" | "release_year" | "score";

/** Sortable table columns, in grid order (checkbox and actions stay fixed). */
const SORTABLE_COLUMNS: { key: SortKey; labelKey: string }[] = [
  { key: "name", labelKey: "common.name" },
  { key: "media_type", labelKey: "common.type" },
  { key: "subtype", labelKey: "common.subtype" },
  { key: "release_year", labelKey: "common.year" },
  { key: "score", labelKey: "common.score" },
];

/** Compares two library items by a sort key. Missing values always sort last. */
function compareItems(a: LibraryItem, b: LibraryItem, key: SortKey): number {
  switch (key) {
    case "name":
      return a.name.localeCompare(b.name, undefined, { sensitivity: "base" });
    case "media_type":
      return a.media_type.localeCompare(b.media_type);
    case "subtype": {
      const sa = a.subtype ?? "";
      const sb = b.subtype ?? "";
      if (sa === sb) return 0;
      if (sa === "") return 1;
      if (sb === "") return -1;
      return sa.localeCompare(sb, undefined, { sensitivity: "base" });
    }
    case "release_year": {
      const ya = a.release_year;
      const yb = b.release_year;
      if (ya === yb) return 0;
      if (ya == null) return 1;
      if (yb == null) return -1;
      return ya - yb;
    }
    case "score": {
      const sa = a.score;
      const sb = b.score;
      if (sa === sb) return 0;
      if (sa == null) return 1;
      if (sb == null) return -1;
      return sa - sb;
    }
  }
}

/** Clickable column header that toggles the table sort on click. */
function SortableHeader({
  label,
  active,
  dir,
  onClick,
}: {
  label: string;
  active: boolean;
  dir: "asc" | "desc";
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={label}
      style={{
        background: "none",
        border: "none",
        padding: 0,
        margin: 0,
        fontFamily: "var(--font-family)",
        fontSize: "var(--font-size-sm)",
        fontWeight: 600,
        color: "inherit",
        cursor: "pointer",
        textAlign: "left",
        display: "inline-flex",
        alignItems: "center",
        gap: "0.3rem",
        whiteSpace: "nowrap",
      }}
    >
      {label}
      <span
        style={{
          fontSize: "0.7em",
          color: active ? "var(--color-primary)" : "var(--color-text-muted)",
        }}
      >
        {active ? (dir === "asc" ? "▲" : "▼") : "↕"}
      </span>
    </button>
  );
}

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

/** One row of the library table: selection checkbox, fields and row actions. */
function LibraryRow({
  item,
  selected,
  onSelect,
  onToggleDone,
  onEdit,
  onDelete,
}: {
  item: LibraryItem;
  selected: boolean;
  onSelect: (checked: boolean) => void;
  onToggleDone: (done: boolean) => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const { t } = useI18n();
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns: LIBRARY_GRID,
        gap: "var(--space-sm)",
        alignItems: "center",
        padding: "0.5rem var(--space-md)",
        borderBottom: "1px solid var(--color-border-light)",
        background: selected ? "var(--color-primary-subtle)" : undefined,
        minWidth: 640,
      }}
    >
      <Checkbox checked={selected} onChange={onSelect} title={t("library.selectBulk")} />
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
        {MEDIA_ICONS[item.media_type]} {t(MEDIA_LABEL_KEYS[item.media_type] ?? "common.type")}
      </span>
      <span className="text-sm" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
        {item.subtype || <span style={{ color: "var(--color-text-muted)" }}>—</span>}
      </span>
      <span className="text-sm">{item.release_year ?? "—"}</span>
      <span className="text-sm" style={{ overflowWrap: "anywhere" }}>
        {item.score != null ? `★ ${formatScore(item.score)}` : "—"}
      </span>
      <div className="flex-center" style={{ gap: "0.35rem" }}>
        <Checkbox
          checked={item.done}
          onChange={onToggleDone}
          title={item.done ? t("library.markNotDone") : t("library.markDone")}
        />
        <button className="btn btn-sm" onClick={onEdit}>{t("common.edit")}</button>
        <button className="btn btn-danger btn-sm" onClick={onDelete}>
          {t("common.delete")}
        </button>
      </div>
    </div>
  );
}

export default function Library() {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [typeFilter, setTypeFilter] = useState("");
  const [doneFilter, setDoneFilter] = useState("");
  const [subtypeFilter, setSubtypeFilter] = useState("");
  // Table sort: default to alphabetical by name.
  const [sort, setSort] = useState<{ key: SortKey; dir: "asc" | "desc" }>({ key: "name", dir: "asc" });
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<LibraryItem | null>(null);
  const [form, setForm] = useState<CreateLibraryItemRequest>(EMPTY_FORM);
  const [importOpen, setImportOpen] = useState(false);
  const [bulkScoreOpen, setBulkScoreOpen] = useState(false);
  const [scoreHits, setScoreHits] = useState<ScoreCandidate[] | null>(null);
  const [scoreSearching, setScoreSearching] = useState(false);
  const [scoreError, setScoreError] = useState<string | null>(null);
  // Ids selected for bulk actions (e.g. delete).
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const confirm = useConfirm();
  // Score lookup is gated by the backend flag `library.score_lookup_enabled`
  // (off by default). Hide the button so users don't hit a 503 "disabled".
  const scoreLookupEnabled = useFeatureFlag("library.score_lookup_enabled");

  const { data: items = [], isLoading, error: listError } = useQuery({
    queryKey: ["library", typeFilter, doneFilter, subtypeFilter],
    queryFn: () =>
      listLibraryItems(
        (typeFilter || undefined) as MediaType | undefined,
        doneFilter === "" ? undefined : doneFilter === "true",
        subtypeFilter || undefined,
      ),
  });

  // Genre/console options for the subtype filter, scoped by the type filter so
  // games show consoles and movies/series/books show genres.
  const { data: subtypeOptions = [] } = useQuery({
    queryKey: ["library-subtypes", typeFilter],
    queryFn: () => listLibrarySubtypes((typeFilter || undefined) as MediaType | undefined),
  });

  // Quick subtype filters, derived from the subtypes that actually exist in the
  // user's items (same philosophy as the recipe tag chips). Each chip pairs the
  // subtype with its media type so e.g. a Horror book is not confused with a
  // Horror movie.
  const { data: allItems = [] } = useQuery({
    queryKey: ["library", "all"],
    queryFn: () => listLibraryItems(),
  });

  const availableSubtypePairs = useMemo(() => {
    const seen = new Set<string>();
    const pairs: { media_type: MediaType; subtype: string }[] = [];
    for (const item of allItems) {
      if (!item.subtype) continue;
      const key = `${item.media_type}:${item.subtype}`;
      if (seen.has(key)) continue;
      seen.add(key);
      pairs.push({ media_type: item.media_type, subtype: item.subtype });
    }
    // Same-named subtypes from different types stay adjacent (so the type icon
    // disambiguation is obvious); within a subtype, order by media type.
    return pairs.sort(
      (a, b) =>
        a.subtype.localeCompare(b.subtype) || a.media_type.localeCompare(b.media_type),
    );
  }, [allItems]);

  const itemById = useMemo(() => new Map(allItems.map((item) => [item.id, item])), [allItems]);

  const scoreTargetItems = useMemo(() => {
    if (selected.size === 0) return [] as LibraryItem[];
    const out: LibraryItem[] = [];
    for (const id of selected) {
      const item = itemById.get(id);
      if (item && item.score == null) out.push(item);
    }
    return out;
  }, [selected, itemById]);

  // Sort the displayed items client-side (the list is fully loaded, no
  // pagination). Missing values are kept at the end regardless of direction.
  const sortedItems = useMemo(() => {
    const dir = sort.dir === "asc" ? 1 : -1;
    return [...items].sort((a, b) => compareItems(a, b, sort.key) * dir);
  }, [items, sort]);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["library"] });

  const create = useMutation({
    mutationFn: () =>
      createLibraryItem({
        ...form,
        score: form.score != null ? normalizeScore(form.score) : null,
      }),
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
        score: form.score != null ? normalizeScore(form.score) : null,
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

  /** Adds or removes one row from the bulk-selection set. */
  function toggleSelected(id: string, checked: boolean) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  }

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
      score: item.score != null ? normalizeScore(item.score) : null,
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

  /** Toggles a quick subtype chip: activates it (setting both the type and the
   *  subtype filters) or, when already active, clears the pair. */
  function toggleSubtypeFilter(mediaType: string, subtype: string) {
    if (typeFilter === mediaType && subtypeFilter === subtype) {
      setTypeFilter("");
      setSubtypeFilter("");
    } else {
      setTypeFilter(mediaType);
      setSubtypeFilter(subtype);
    }
  }

  /** Toggles a column sort: re-sorting the same column flips the direction. */
  function toggleSort(key: SortKey) {
    setSort((prev) =>
      prev.key === key ? { key, dir: prev.dir === "asc" ? "desc" : "asc" } : { key, dir: "asc" },
    );
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
      if (hits.length === 0) setScoreError(t("library.noRatings"));
    } catch (err) {
      setScoreError(err instanceof Error ? err.message : String(err));
    } finally {
      setScoreSearching(false);
    }
  }

  /** Applies a confirmed candidate's score + source to the form. The score is
   *  truncated to one decimal so the input shows exactly what will be saved. */
  function applyCandidate(candidate: ScoreCandidate) {
    setForm({
      ...form,
      score: normalizeScore(candidate.score),
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
        <h2 className="page-heading" style={{ marginBottom: 0 }}><FilmIcon size={24} /> {t("library.title")}</h2>
      </div>

      {/* Toolbar: add / import / filters */}
      <div className="flex-center" style={{ gap: "var(--space-sm)", marginBottom: "var(--space-md)", flexWrap: "wrap" }}>
        <button className="btn btn-primary" onClick={openAdd} disabled={formOpen}>
          {formOpen ? t("library.editing") : t("library.addItem")}
        </button>
        <button className="btn" onClick={() => setImportOpen(true)}>📥 {t("library.importCsv")}</button>
        <select
          className="select"
          value={typeFilter}
          onChange={(e) => {
            setTypeFilter(e.target.value);
            // The subtype list is scoped by media type. Drop a stale value
            // (e.g. a console filter while browsing movies).
            setSubtypeFilter("");
          }}
          aria-label={t("library.filterType")}
        >
          <option value="">{t("library.allTypes")}</option>
          {MEDIA_TYPES.map((type) => (
            <option key={type} value={type}>
              {MEDIA_ICONS[type]} {t(MEDIA_LABEL_KEYS[type])}
            </option>
          ))}
        </select>
        <select
          className="select"
          value={doneFilter}
          onChange={(e) => setDoneFilter(e.target.value)}
          aria-label={t("library.filterStatus")}
        >
          <option value="">{t("common.all")}</option>
          <option value="true">{t("common.done")}</option>
          <option value="false">{t("common.notDone")}</option>
        </select>
        <select
          className="select"
          value={subtypeFilter}
          onChange={(e) => setSubtypeFilter(e.target.value)}
          aria-label={t("library.filterSubtype")}
        >
          <option value="">{t("library.filterSubtypeAll")}</option>
          {subtypeOptions.map((subtype) => (
            <option key={subtype} value={subtype}>
              {subtype}
            </option>
          ))}
        </select>
      </div>

      {/* Quick subtype filters, derived from the subtypes that actually exist.
          Each chip pairs the subtype with its media type so e.g. a Horror book
          is not confused with a Horror movie. Clicking one narrows the list;
          clicking it again clears the filter. */}
      {availableSubtypePairs.length > 0 && (
        <div className="flex-center" style={{ gap: "0.4rem", flexWrap: "wrap", marginBottom: "var(--space-lg)" }}>
          <span className="text-sm text-secondary" style={{ fontWeight: 600 }}>
            {t("library.quickFilters")}:
          </span>
          {availableSubtypePairs.map(({ media_type, subtype }) => {
            const active = typeFilter === media_type && subtypeFilter === subtype;
            return (
              <button
                key={`${media_type}:${subtype}`}
                className={`badge ${active ? "badge-done" : "badge-habit"}`}
                style={{ cursor: "pointer", border: "none", fontFamily: "var(--font-family)" }}
                title={`${t(MEDIA_LABEL_KEYS[media_type])} · ${subtype}`}
                onClick={() => toggleSubtypeFilter(media_type, subtype)}
              >
                {MEDIA_ICONS[media_type]} {subtype}
              </button>
            );
          })}
        </div>
      )}

      {/* Add / edit item modal */}
      {formOpen && (
        <Modal onClose={closeForm} maxWidth={560}>
          <h3 className="modal-title">
            {editing ? t("library.editItemTitle") : t("library.addItemTitle")}
          </h3>
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(160px, 1fr))", gap: "0.4rem", marginBottom: "0.4rem" }}>
            <input
              className="input"
              style={{ gridColumn: "1 / -1" }}
              placeholder={t("library.namePlaceholder")}
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              autoFocus
            />
            <select
              className="select"
              value={form.media_type}
              onChange={(e) => setForm({ ...form, media_type: e.target.value as MediaType })}
            >
              {MEDIA_TYPES.map((type) => (
                <option key={type} value={type}>
                  {MEDIA_ICONS[type]} {t(MEDIA_LABEL_KEYS[type])}
                </option>
              ))}
            </select>
            <input
              className="input"
              type="number"
              min={1800}
              max={2100}
              placeholder={t("library.yearPlaceholder")}
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
              placeholder={form.media_type === "game" ? t("library.consolePlaceholder") : t("library.genrePlaceholder")}
              value={form.subtype}
              onChange={(e) => setForm({ ...form, subtype: e.target.value })}
            />
            <textarea
              className="input"
              style={{ gridColumn: "1 / -1", minHeight: 56 }}
              placeholder={t("library.notesPlaceholder")}
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
              placeholder={t("library.scorePlaceholder")}
              value={form.score ?? ""}
              onChange={(e) =>
                setForm({ ...form, score: e.target.value === "" ? null : Number(e.target.value) })
              }
              style={{ flex: 1, minWidth: 120 }}
            />
            <input
              className="input"
              placeholder={t("library.sourcePlaceholder")}
              value={form.score_source}
              onChange={(e) => setForm({ ...form, score_source: e.target.value })}
              style={{ flex: 1, minWidth: 140 }}
            />
            {scoreLookupEnabled && (
              <button
                type="button"
                className="btn btn-sm"
                disabled={!form.name.trim() || scoreSearching}
                onClick={lookUpScore}
              >
                {scoreSearching ? t("library.searching") : t("library.lookUpScore")}
              </button>
            )}
          </div>

          {scoreError && (
            <p className="text-sm" style={{ color: "var(--color-danger)", margin: "0 0 0.4rem" }}>
              {scoreError}
            </p>
          )}

          {scoreHits && scoreHits.length > 0 && (
            <div style={{ marginBottom: "0.4rem" }}>
              <p className="text-sm" style={{ fontWeight: 600 }}>{t("library.pickMatch")}:</p>
              {scoreHits.map((hit) => (
                <button
                  key={hit.external_id || hit.title}
                  type="button"
                  className="btn btn-sm"
                  style={{ margin: "0 0.25rem 0.25rem 0" }}
                  onClick={() => applyCandidate(hit)}
                >
                  {hit.title}
                  {hit.year != null && hit.year > 0 ? ` (${hit.year})` : ""} · ★ {formatScore(hit.score)}
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
            <span className="text-sm">{t("library.doneLabel")}</span>
          </label>
          {(create.error ?? update.error) && (
            <p className="text-sm" style={{ color: "var(--color-danger)", margin: "0 0 0.4rem" }}>
              {String(create.error ?? update.error)}
            </p>
          )}
          <div className="modal-actions">
            <button
              className="btn btn-primary"
              disabled={!form.name.trim() || saving}
              onClick={save}
            >
              {saving ? t("common.saving") : editing ? t("library.saveChanges") : t("common.add")}
            </button>
            <button className="btn btn-ghost" onClick={closeForm}>
              {t("common.cancel")}
            </button>
          </div>
        </Modal>
      )}

      {!formOpen && saveError && <p style={{ color: "var(--color-danger)" }}>{String(saveError)}</p>}
      {listError && <p style={{ color: "var(--color-danger)" }}>{t("library.loadFailed")}</p>}
      {isLoading && <p style={{ color: "var(--color-text-secondary)" }}>{t("library.loading")}</p>}
      {items.length === 0 && !isLoading && (
        <div className="empty-state">
          <div className="empty-state-icon">🎬</div>
          <p className="empty-state-text">
            {t("library.empty")}
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
          <span className="text-sm text-secondary">{t("library.selectedCount", { count: selected.size })}</span>
          {scoreLookupEnabled && (
            <button
              className="btn btn-sm"
              title={t("library.bulkScoreHint")}
              disabled={scoreTargetItems.length === 0}
              onClick={() => setBulkScoreOpen(true)}
            >
              {t("library.bulkScoreTitle")}
              {scoreTargetItems.length > 0 ? ` (${scoreTargetItems.length})` : ""}
            </button>
          )}
          <button
            className="btn btn-danger btn-sm"
            disabled={bulkDelete.isPending}
            onClick={async () => {
              const ok = await confirm({
                title: t("library.deleteSelectedConfirm", { count: selected.size }),
                message: t("library.cannotUndo"),
                confirmLabel: t("common.delete"),
                confirmVariant: "danger",
              });
              if (ok) bulkDelete.mutate([...selected]);
            }}
          >
            {bulkDelete.isPending ? t("common.deleting") : t("library.deleteSelected", { count: selected.size })}
          </button>
          <button className="btn btn-ghost btn-sm" onClick={() => setSelected(new Set())}>
            {t("library.clear")}
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
            gridTemplateColumns: LIBRARY_GRID,
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
          <Checkbox
            checked={items.length > 0 && selected.size === items.length}
            onChange={(checked) =>
              setSelected(checked ? new Set(items.map((i) => i.id)) : new Set())
            }
            title={t("library.selectAll")}
          />
          {SORTABLE_COLUMNS.map((col) => (
            <SortableHeader
              key={col.key}
              label={t(col.labelKey)}
              active={sort.key === col.key}
              dir={sort.dir}
              onClick={() => toggleSort(col.key)}
            />
          ))}
          <span>{t("common.actions")}</span>
        </div>

        {sortedItems.map((item) => (
          <LibraryRow
            key={item.id}
            item={item}
            selected={selected.has(item.id)}
            onSelect={(checked) => toggleSelected(item.id, checked)}
            onToggleDone={(done) => toggleDone.mutate({ id: item.id, done })}
            onEdit={() => openEdit(item)}
            onDelete={() => del.mutate(item.id)}
          />
        ))}
      </div>

      {importOpen && (
        <LibraryCsvImport
          onClose={() => setImportOpen(false)}
          onImported={() => {
            invalidate();
            // The import is done — close the modal so the refreshed list is visible.
            setImportOpen(false);
          }}
        />
      )}

      {bulkScoreOpen && scoreTargetItems.length > 0 && (
        <LibraryBulkScore
          items={scoreTargetItems}
          skippedCount={selected.size - scoreTargetItems.length}
          onSaved={() => {
            invalidate();
            setBulkScoreOpen(false);
            setSelected(new Set());
          }}
          onClose={() => setBulkScoreOpen(false)}
        />
      )}
    </div>
  );
}

