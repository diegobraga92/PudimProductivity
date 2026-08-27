import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import {
  importLibraryItems,
  searchLibraryScoresBatch,
  type CreateLibraryItemRequest,
  type ImportResult,
} from "../api/library";
import { applyAutoScoreResults, parseCsv, parseDoneValue, parseScoreValue, normalizeMediaType, parseYearValue, type MediaTypeValue } from "../utils/csv";
import { useFeatureFlag } from "../hooks/useFeatureFlag";
import { useI18n } from "../i18n";

const MEDIA_TYPES = ["movie", "series", "book", "game"] as const;

type FieldKey = "name" | "media_type" | "release_year" | "done" | "notes" | "score" | "score_source" | "subtype";

const FIELDS: { key: FieldKey; labelKey: string; hintKey: string }[] = [
  { key: "name", labelKey: "csv.fieldName", hintKey: "csv.fieldNameHint" },
  { key: "media_type", labelKey: "csv.fieldType", hintKey: "csv.fieldTypeHint" },
  { key: "release_year", labelKey: "csv.fieldYear", hintKey: "csv.fieldYearHint" },
  { key: "done", labelKey: "csv.fieldDone", hintKey: "csv.fieldDoneHint" },
  { key: "score", labelKey: "csv.fieldScore", hintKey: "csv.fieldScoreHint" },
  { key: "score_source", labelKey: "csv.fieldSource", hintKey: "csv.fieldSourceHint" },
  { key: "subtype", labelKey: "csv.fieldSubtype", hintKey: "csv.fieldSubtypeHint" },
  { key: "notes", labelKey: "csv.fieldNotes", hintKey: "csv.fieldNotesHint" },
];

const FIXED_VALUE = "__fixed__";

const EMPTY_MAPPING: Record<FieldKey, string> = {
  name: "",
  media_type: "",
  release_year: "",
  done: "",
  notes: "",
  score: "",
  score_source: "",
  subtype: "",
};

const DEFAULT_FIXED: Record<FieldKey, string> = {
  name: "",
  media_type: "book",
  release_year: "",
  done: "false",
  notes: "",
  score: "",
  score_source: "",
  subtype: "",
};

/** Header-based auto-mapping: matches common column names case-insensitively. */
function suggestMapping(headers: string[]): Record<FieldKey, string> {
  const slug = headers.map((h) => h.toLowerCase().replace(/[^a-z0-9]+/g, ""));
  const find = (aliases: string[]): string => {
    const idx = slug.findIndex((h) => aliases.includes(h));
    return idx === -1 ? "" : String(idx);
  };
  return {
    name: find(["name", "title"]),
    media_type: find(["type", "mediatype", "kind"]),
    release_year: find(["year", "releaseyear", "yearreleased", "yearofrelease"]),
    done: find(["done", "status", "completed", "finished", "watched", "read", "played"]),
    score: find(["score", "rating", "metascore", "metacritic", "imdbrating"]),
    score_source: find(["scoresource", "ratingsource", "source", "site"]),
    notes: find(["notes", "note", "comments", "comment"]),
    subtype: find(["subtype", "genre", "console", "platform", "category"]),
  };
}

interface LibraryCsvImportProps {
  onClose: () => void;
  onImported: (result: ImportResult) => void;
}

/**
 * CSV import dialog with column matching + fixed values. Parses the file
 * client-side, lets the user map each field to a column (or a constant), shows
 * a preview, then POSTs the resolved items to /library/import.
 */
export function LibraryCsvImport({ onClose, onImported }: LibraryCsvImportProps) {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const fileRef = useRef<HTMLInputElement>(null);

  const [fileName, setFileName] = useState("");
  const [rows, setRows] = useState<string[][]>([]);
  const [hasHeader, setHasHeader] = useState(true);
  const [mapping, setMapping] = useState<Record<FieldKey, string>>(EMPTY_MAPPING);
  const [fixed, setFixed] = useState<Record<FieldKey, string>>(DEFAULT_FIXED);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);

  // Auto-scoring (critic score lookup during import).
  type AutoFill = { score: number; score_source: string; release_year?: number };
  const [autoFilled, setAutoFilled] = useState<Record<number, AutoFill>>({});
  const [autoScoring, setAutoScoring] = useState(false);
  const [autoScoreError, setAutoScoreError] = useState<string | null>(null);
  const [autoScoredCount, setAutoScoredCount] = useState(0);
  const [noRatingCount, setNoRatingCount] = useState(0);
  const scoreLookupEnabled = useFeatureFlag("library.score_lookup_enabled");

  const headers = hasHeader && rows.length > 0 ? rows[0] : rows[0]?.map((_, i) => `Column ${i + 1}`);
  const dataRows = hasHeader && rows.length > 0 ? rows.slice(1) : rows;

  const mutation = useMutation({
    mutationFn: (items: CreateLibraryItemRequest[]) =>
      importLibraryItems({ items }).then((res) => {
        queryClient.invalidateQueries({ queryKey: ["library"] });
        return res;
      }),
    onSuccess: (res) => {
      setResult(res);
      onImported(res);
    },
    onError: (err: Error) => setError(err.message),
  });

  function handleFile(file: File | undefined) {
    if (!file) return;
    setError(null);
    setResult(null);
    setFileName(file.name);
    const reader = new FileReader();
    reader.onload = () => {
      const text = String(reader.result ?? "");
      const parsed = parseCsv(text);
      if (parsed.length === 0) {
        setError(t("csv.emptyFile"));
        setRows([]);
        return;
      }
      setRows(parsed);
      setMapping(suggestMapping(parsed[0]));
    };
    reader.onerror = () => setError(t("csv.readError"));
    reader.readAsText(file);
  }

  /** Resolves the effective value for a field at a data-row index. Auto-scored
   *  values override whatever the column mapping produced. */
  function resolve(field: FieldKey, rowIndex: number): string {
    const filled = autoFilled[rowIndex];
    if (field === "score" && filled?.score != null) return String(filled.score);
    if (field === "score_source" && filled?.score_source) return filled.score_source;
    if (field === "release_year" && filled?.release_year != null) return String(filled.release_year);
    const m = mapping[field];
    if (m === FIXED_VALUE) return fixed[field];
    if (m === "" || m === null) return "";
    const col = Number(m);
    const cell = dataRows[rowIndex]?.[col];
    return cell === undefined ? "" : cell;
  }

  /**
   * Builds the import payload, collecting per-row validation errors. Auto-scored
   * values are read from the committed `autoFilled` state via `resolve`.
   */
  function buildItems(): { items: CreateLibraryItemRequest[]; rowErrors: string[] } {
    const items: CreateLibraryItemRequest[] = [];
    const rowErrors: string[] = [];
    dataRows.forEach((_, i) => {
      const rowNum = i + 1;
      const name = resolve("name", i).trim();
      if (!name) {
        rowErrors.push(t("csv.rowMissingName", { row: rowNum }));
        return;
      }
      const mediaRaw = resolve("media_type", i);
      const mediaType = normalizeMediaType(mediaRaw);
      if (!mediaType) {
        rowErrors.push(t("csv.rowInvalidType", { row: rowNum, type: mediaRaw.trim() }));
        return;
      }
      items.push({
        name,
        media_type: mediaType,
        release_year: parseYearValue(resolve("release_year", i)),
        done: parseDoneValue(resolve("done", i)),
        score: parseScoreValue(resolve("score", i)),
        score_source: resolve("score_source", i).trim(),
        subtype: resolve("subtype", i).trim(),
        notes: resolve("notes", i),
      });
    });
    return { items, rowErrors };
  }

  /**
   * Looks up critic scores for every row that has no score yet, using the batch
   * score endpoint in chunks of 100. Returns the auto-fill map (data-row index
   * → score + source + release year) plus simple stats for the UI.
   */
  async function runAutoScore(): Promise<{ fill: Record<number, AutoFill>; found: number; noRating: number }> {
    const targets: { row: number; name: string; type: MediaTypeValue; year?: number }[] = [];
    dataRows.forEach((_, i) => {
      const name = resolve("name", i).trim();
      const mediaRaw = resolve("media_type", i);
      const mediaType = normalizeMediaType(mediaRaw);
      if (!name || !mediaType) return;
      if (resolve("score", i) !== "") return; // already has a score — keep it
      const year = parseYearValue(resolve("release_year", i));
      targets.push({ row: i, name, type: mediaType, year: year ?? undefined });
    });

    const fill: Record<number, AutoFill> = {};
    let found = 0;
    let noRating = 0;
    for (let start = 0; start < targets.length; start += 100) {
      const chunk = targets.slice(start, start + 100);
      const resp = await searchLibraryScoresBatch(
        chunk.map((c) => ({ name: c.name, type: c.type, year: c.year })),
      );
      const applied = applyAutoScoreResults(
        chunk.map((_, idx) => ({ requestIndex: idx, row: chunk[idx].row })),
        resp.results,
      );
      Object.assign(fill, applied.fill);
      found += applied.found;
      noRating += applied.noRating;
    }
    return { fill, found, noRating };
  }

  /** Auto-scores the preview so the user can review before importing. */
  async function autoScoreAndApply() {
    if (!scoreLookupEnabled) {
      setAutoScoreError(t("csv.autoScoreUnavailable"));
      return;
    }
    setAutoScoring(true);
    setAutoScoreError(null);
    try {
      const out = await runAutoScore();
      setAutoFilled(out.fill);
      setAutoScoredCount(out.found);
      setNoRatingCount(out.noRating);
    } catch (e) {
      setAutoScoreError(e instanceof Error ? e.message : String(e));
    } finally {
      setAutoScoring(false);
    }
  }

  /** Imports the current preview as-is. Auto-scoring is a separate, explicit
   *  step (the "Auto-score" button) so the user can review the filled values in
   *  the preview before committing. */
  function handleImport() {
    setError(null);
    const { items } = buildItems();
    if (items.length === 0) return;
    mutation.mutate(items);
  }

  const preview = buildItems();
  const previewRows = dataRows.slice(0, 5);
  const columns = headers ?? [];

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div
        className="modal-dialog animate-fade-in"
        style={{ maxWidth: 720, width: "100%" }}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="csv-import-title"
      >
        <h3 id="csv-import-title" className="modal-title">
          {t("csv.title")}
        </h3>

        <input
          ref={fileRef}
          type="file"
          accept=".csv,text/csv"
          style={{ display: "none" }}
          onChange={(e) => handleFile(e.target.files?.[0])}
        />

        {rows.length === 0 ? (
          <>
            <p className="modal-message">
              {t("csv.intro")}
            </p>
            <div className="flex-center" style={{ gap: "var(--space-sm)", justifyContent: "flex-start" }}>
              <button className="btn btn-primary" onClick={() => fileRef.current?.click()}>
                {t("csv.chooseFile")}
              </button>
              <button className="btn btn-ghost" onClick={onClose}>
                {t("common.cancel")}
              </button>
            </div>
          </>
        ) : (
          <>
            <p className="text-sm text-secondary" style={{ margin: "0 0 var(--space-sm)" }}>
              📄 {fileName} · {t("csv.fileInfo", { count: dataRows.length })}
            </p>

            <label className="flex-center" style={{ gap: "var(--space-sm)", marginBottom: "var(--space-sm)" }}>
              <input
                type="checkbox"
                checked={hasHeader}
                onChange={(e) => setHasHeader(e.target.checked)}
              />
              <span className="text-sm">{t("csv.firstRowHeader")}</span>
            </label>

            {/* Column mapping */}
            <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(210px, 1fr))", gap: "var(--space-sm)", marginBottom: "var(--space-sm)" }}>
              {FIELDS.map((f) => (
                <div key={f.key}>
                  <label className="text-sm" style={{ fontWeight: 600 }}>
                    {t(f.labelKey)}
                  </label>
                  <select
                    className="select"
                    style={{ width: "100%" }}
                    value={mapping[f.key]}
                    onChange={(e) => setMapping({ ...mapping, [f.key]: e.target.value })}
                  >
                    <option value="">{t("csv.notMapped")}</option>
                    {columns.map((c, i) => (
                      <option key={i} value={String(i)}>
                        {c || t("csv.column", { number: i + 1 })}
                      </option>
                    ))}
                    {f.key !== "name" && (
                      <option value={FIXED_VALUE}>{t("csv.fixedValue")}</option>
                    )}
                  </select>
                  {mapping[f.key] === FIXED_VALUE && (
                    <div className="flex-center" style={{ gap: "0.25rem", marginTop: "0.25rem" }}>
                      {f.key === "media_type" ? (
                        <select
                          className="select"
                          value={fixed.media_type}
                          onChange={(e) => setFixed({ ...fixed, media_type: e.target.value })}
                        >
                          {MEDIA_TYPES.map((type) => (
                            <option key={type} value={type}>
                              {type}
                            </option>
                          ))}
                        </select>
                      ) : f.key === "done" ? (
                        <select
                          className="select"
                          value={fixed.done}
                          onChange={(e) => setFixed({ ...fixed, done: e.target.value })}
                        >
                          <option value="false">{t("common.notDone")}</option>
                          <option value="true">{t("common.done")}</option>
                        </select>
                      ) : (
                        <input
                          className="input"
                          style={{ width: "100%" }}
                          placeholder={t(f.labelKey)}
                          value={fixed[f.key]}
                          onChange={(e) => setFixed({ ...fixed, [f.key]: e.target.value })}
                        />
                      )}
                    </div>
                  )}
                  <p className="text-sm text-secondary" style={{ margin: "0.15rem 0 0" }}>
                    {t(f.hintKey)}
                  </p>
                </div>
              ))}
            </div>

            {/* Auto-score: batch critic-score lookup for rows without a score */}
            <div className="flex-center" style={{ gap: "0.5rem", marginBottom: "var(--space-sm)", flexWrap: "wrap" }}>
              <button
                className="btn"
                disabled={autoScoring || preview.items.length === 0}
                onClick={autoScoreAndApply}
              >
                {autoScoring ? t("csv.autoScoring") : t("csv.autoScore")}
              </button>
              {autoScoredCount > 0 && (
                <span className="text-sm" style={{ color: "var(--color-done)" }}>
                  {t("csv.autoScored", { count: autoScoredCount })}
                </span>
              )}
              {noRatingCount > 0 && (
                <span className="text-sm text-secondary">
                  {noRatingCount}× {t("csv.noRating")}
                </span>
              )}
              {autoScoreError && (
                <span className="text-sm" style={{ color: "var(--color-danger)" }}>
                  {autoScoreError}
                </span>
              )}
            </div>


            {/* Preview */}
            {previewRows.length > 0 && (
              <div style={{ marginBottom: "var(--space-sm)", overflowX: "auto" }}>
                <p className="text-sm" style={{ fontWeight: 600 }}>{t("csv.preview")}</p>
                <table style={{ width: "100%", borderCollapse: "collapse", fontSize: "var(--font-size-xs)" }}>
                  <thead>
                    <tr>
                      {FIELDS.map((f) => (
                        <th key={f.key} style={{ textAlign: "left", padding: "0.25rem 0.5rem", borderBottom: "1px solid var(--color-border, #ddd)" }}>
                          {t(f.labelKey)}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {previewRows.map((_, i) => (
                      <tr key={i}>
                        {FIELDS.map((f) => (
                          <td key={f.key} style={{ padding: "0.25rem 0.5rem", borderBottom: "1px solid var(--color-border, #ddd)" }}>
                            {resolve(f.key, i) || "—"}
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {preview.rowErrors.length > 0 && (
              <div style={{ marginBottom: "var(--space-sm)" }}>
                <p className="text-sm" style={{ color: "var(--color-danger)", fontWeight: 600 }}>
                  {t("csv.rowsSkipped", { count: preview.rowErrors.length })}
                </p>
                <ul style={{ margin: 0, paddingLeft: "1.1rem", fontSize: "var(--font-size-xs)", color: "var(--color-danger)" }}>
                  {preview.rowErrors.slice(0, 10).map((e, i) => (
                    <li key={i}>{e}</li>
                  ))}
                  {preview.rowErrors.length > 10 && <li>{t("csv.andMore", { count: preview.rowErrors.length - 10 })}</li>}
                </ul>
              </div>
            )}

            {result && (
              <p className="text-sm" style={{ color: "var(--color-done)", marginBottom: "var(--space-sm)" }}>
                ✅ {t("csv.imported", { count: result.imported })}
                {result.errors && result.errors.length > 0 && ` · ${t("csv.rowsSkippedByServer", { count: result.errors.length })}`}
              </p>
            )}
            {error && <p style={{ color: "var(--color-danger)", marginBottom: "var(--space-sm)" }}>{error}</p>}

            <div className="modal-actions">
              <button className="btn btn-ghost" onClick={() => fileRef.current?.click()}>
                {t("csv.chooseAnother")}
              </button>
              <button className="btn btn-ghost" onClick={onClose}>
                {t("common.close")}
              </button>
              <button
                className="btn btn-primary"
                disabled={mutation.isPending || autoScoring || preview.items.length === 0}
                onClick={handleImport}
              >
                {mutation.isPending
                  ? t("common.importing")
                  : autoScoring
                    ? t("csv.autoScoring")
                    : t("csv.importCount", { count: preview.items.length })}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

