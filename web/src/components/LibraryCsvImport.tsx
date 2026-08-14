import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import {
  importLibraryItems,
  type CreateLibraryItemRequest,
  type ImportResult,
} from "../api/library";
import { parseCsv, parseDoneValue, parseScoreValue, normalizeMediaType, parseYearValue } from "../utils/csv";
import { useI18n } from "../i18n";

const MEDIA_TYPES = ["movie", "series", "book", "game"] as const;

type FieldKey = "name" | "media_type" | "release_year" | "done" | "notes" | "score" | "score_source";

const FIELDS: { key: FieldKey; labelKey: string; hintKey: string }[] = [
  { key: "name", labelKey: "csv.fieldName", hintKey: "csv.fieldNameHint" },
  { key: "media_type", labelKey: "csv.fieldType", hintKey: "csv.fieldTypeHint" },
  { key: "release_year", labelKey: "csv.fieldYear", hintKey: "csv.fieldYearHint" },
  { key: "done", labelKey: "csv.fieldDone", hintKey: "csv.fieldDoneHint" },
  { key: "score", labelKey: "csv.fieldScore", hintKey: "csv.fieldScoreHint" },
  { key: "score_source", labelKey: "csv.fieldSource", hintKey: "csv.fieldSourceHint" },
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
};

const DEFAULT_FIXED: Record<FieldKey, string> = {
  name: "",
  media_type: "book",
  release_year: "",
  done: "false",
  notes: "",
  score: "",
  score_source: "",
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

  /** Resolves the effective value for a field at a data-row index. */
  function resolve(field: FieldKey, rowIndex: number): string {
    const m = mapping[field];
    if (m === FIXED_VALUE) return fixed[field];
    if (m === "" || m === null) return "";
    const col = Number(m);
    const cell = dataRows[rowIndex]?.[col];
    return cell === undefined ? "" : cell;
  }

  /** Builds the import payload, collecting per-row validation errors. */
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
        notes: resolve("notes", i),
      });
    });
    return { items, rowErrors };
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
                disabled={mutation.isPending || preview.items.length === 0}
                onClick={() => mutation.mutate(preview.items)}
              >
                {mutation.isPending
                  ? t("common.importing")
                  : t("csv.importCount", { count: preview.items.length })}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

