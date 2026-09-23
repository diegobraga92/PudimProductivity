import { useCallback, useEffect, useState } from "react";
import {
  searchLibraryScoresBatch,
  updateLibraryItem,
  type LibraryItem,
  type ScoreCandidate,
} from "../api/library";
import { useI18n } from "../i18n";
import { useToast } from "./toastContext";

const MAX_BATCH_LOOKUP = 100;

const MEDIA_ICONS: Record<string, string> = {
  movie: "🎬",
  series: "📺",
  book: "📚",
  game: "🎮",
};

/** Rounds a score to one decimal (matches the Library list display). */
const roundScore = (score: number): number => Math.round(score * 10) / 10;

/** Human-friendly label for a candidate, e.g. "Alien (1979) · ★ 8.5 (imdb)". */
function candidateLabel(candidate: ScoreCandidate): string {
  const year = candidate.year && candidate.year > 0 ? ` (${candidate.year})` : "";
  const source = candidate.score_source ? ` (${candidate.score_source})` : "";
  return `${candidate.title}${year} · ★ ${roundScore(candidate.score)}${source}`;
}

interface LibraryBulkScoreProps {
  /** Selected items that have no score yet (lookup + save targets). */
  items: LibraryItem[];
  /** How many selected items already had a score and were therefore skipped. */
  skippedCount: number;
  /** Called after at least one score is saved so the list can refresh. */
  onSaved: () => void;
  onClose: () => void;
}

/**
 * Bulk score dialog for the Library.
 */
export function LibraryBulkScore({ items, skippedCount, onSaved, onClose }: LibraryBulkScoreProps) {
  const { t } = useI18n();
  const { pushToast } = useToast();

  // Snapshot the targets once.
  const [targets] = useState<LibraryItem[]>(() => items);

  const [searching, setSearching] = useState(true);
  const [lookupFailed, setLookupFailed] = useState(false);
  const [matches, setMatches] = useState<Record<string, ScoreCandidate[]>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [chosen, setChosen] = useState<Record<string, number>>({});
  const [saving, setSaving] = useState(false);
  const [savedCount, setSavedCount] = useState(0);
  const [saveErrors, setSaveErrors] = useState<string[]>([]);

  const busy = searching || saving;

  /** Fetches score candidates for every target. Pure: it touches no React state,
   * so the mount effect below triggers no synchronous re-render. */
  const fetchCandidates = useCallback(async () => {
    const found: Record<string, ScoreCandidate[]> = {};
    const failed: Record<string, string> = {};
    for (let start = 0; start < targets.length; start += MAX_BATCH_LOOKUP) {
      const chunk = targets.slice(start, start + MAX_BATCH_LOOKUP);
      const resp = await searchLibraryScoresBatch(
        chunk.map((item) => ({
          name: item.name,
          type: item.media_type,
          year: item.release_year ?? null,
        })),
      );
      for (const result of resp.results) {
        const item = chunk[result.index];
        if (!item) continue;
        if (result.error) failed[item.id] = result.error;
        else if (result.candidates.length > 0) found[item.id] = result.candidates;
      }
    }
    return { found, failed };
  }, [targets]);

  /** Applies a finished lookup to the dialog state. */
  const applyLookupResult = useCallback(
    (found: Record<string, ScoreCandidate[]>, failed: Record<string, string>) => {
      setMatches(found);
      setErrors(failed);
      const initial: Record<string, number> = {};
      for (const id of Object.keys(found)) initial[id] = 0;
      setChosen(initial);
    },
    [],
  );

  // Start the lookup as soon as the dialog opens.
  useEffect(() => {
    let cancelled = false;
    fetchCandidates()
      .then(({ found, failed }) => {
        if (!cancelled) applyLookupResult(found, failed);
      })
      .catch(() => {
        if (!cancelled) setLookupFailed(true);
      })
      .finally(() => {
        if (!cancelled) setSearching(false);
      });
    return () => {
      cancelled = true;
    };
  }, [applyLookupResult, fetchCandidates]);

  /** Clears the previous results and starts a fresh lookup (Retry button). */
  const retryLookup = useCallback(() => {
    setSearching(true);
    setLookupFailed(false);
    setMatches({});
    setErrors({});
    setChosen({});
    setSavedCount(0);
    setSaveErrors([]);
    void fetchCandidates()
      .then(({ found, failed }) => applyLookupResult(found, failed))
      .catch(() => setLookupFailed(true))
      .finally(() => setSearching(false));
  }, [applyLookupResult, fetchCandidates]);

  // Close on Escape and lock body scroll while the dialog is open.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleKeyDown);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "";
    };
  }, [onClose]);

  /** Saves the chosen candidate's score + source for every matched item. */
  async function handleSave() {
    const pending = targets.filter((item) => (matches[item.id]?.length ?? 0) > 0);
    if (pending.length === 0 || saving) return;
    setSaving(true);
    setSaveErrors([]);
    let ok = 0;
    const failures: string[] = [];
    for (const item of pending) {
      const candidate = matches[item.id]?.[chosen[item.id] ?? 0];
      if (!candidate) continue;
      try {
        await updateLibraryItem(item.id, {
          score: roundScore(candidate.score),
          score_source: candidate.score_source,
        });
        ok++;
      } catch (err) {
        failures.push(`${item.name}: ${err instanceof Error ? err.message : String(err)}`);
      }
    }
    setSavedCount(ok);
    setSaveErrors(failures);
    setSaving(false);
    if (ok > 0) {
      if (failures.length === 0) {
        pushToast({ icon: "⭐", title: t("library.bulkScoreSaved", { count: ok }) });
      } else {
        pushToast({
          icon: "⚠️",
          title: t("library.bulkScoreSaveError"),
          body: t("library.bulkScoreNotSavedBody", { count: failures.length }),
        });
      }
      onSaved();
    }
  }

  const matchedCount = Object.keys(matches).length;
  const errorCount = Object.keys(errors).length;
  const noRatingCount = Math.max(0, targets.length - matchedCount - errorCount);
  const saveCount = matchedCount;
  const fullySaved = savedCount > 0 && saveErrors.length === 0;

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div
        className="modal-dialog animate-fade-in"
        style={{ maxWidth: 720, width: "100%" }}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="bulk-score-title"
      >
        <h3 id="bulk-score-title" className="modal-title">
          {t("library.bulkScoreTitle")}
        </h3>

        <p className="modal-message">{t("library.bulkScoreIntro")}</p>
        {skippedCount > 0 && (
          <p className="text-sm text-secondary" style={{ margin: "0 0 var(--space-sm)" }}>
            {t("library.bulkScoreSkipped", { count: skippedCount })}
          </p>
        )}

        {targets.length === 0 ? (
          <>
            <p className="text-sm text-secondary">{t("library.bulkScoreEmpty")}</p>
            <div className="modal-actions">
              <button className="btn btn-primary" onClick={onClose}>
                {t("common.close")}
              </button>
            </div>
          </>
        ) : searching ? (
          <>
            <p className="text-sm text-secondary" style={{ padding: "var(--space-md) 0" }}>
              {t("library.bulkScoreLookingUp")}
            </p>
            <div className="modal-actions">
              <button className="btn btn-ghost" onClick={onClose}>
                {t("common.cancel")}
              </button>
            </div>
          </>
        ) : lookupFailed ? (
          <>
            <p className="text-sm" style={{ color: "var(--color-danger)", margin: "0 0 var(--space-sm)" }}>
              {t("library.bulkScoreLookupFailed")}
            </p>
            <div className="modal-actions">
              <button className="btn btn-ghost" onClick={onClose}>
                {t("common.close")}
              </button>
              <button className="btn" onClick={retryLookup}>
                {t("common.retry")}
              </button>
            </div>
          </>
        ) : (
          <>
            <div
              className="flex-center"
              style={{ gap: "0.6rem", flexWrap: "wrap", marginBottom: "var(--space-sm)" }}
            >
              {matchedCount > 0 && (
                <span className="text-sm" style={{ color: "var(--color-done)" }}>
                  {t("library.bulkScoreMatched", { count: matchedCount })}
                </span>
              )}
              {noRatingCount > 0 && (
                <span className="text-sm text-secondary">
                  {t("library.bulkScoreNoRating", { count: noRatingCount })}
                </span>
              )}
              {errorCount > 0 && (
                <span className="text-sm" style={{ color: "var(--color-danger)" }}>
                  {t("library.bulkScoreErrors", { count: errorCount })}
                </span>
              )}
            </div>

            <div
              style={{
                display: "flex",
                flexDirection: "column",
                gap: "0.4rem",
                maxHeight: "38vh",
                overflowY: "auto",
                border: "1px solid var(--color-border)",
                borderRadius: "var(--radius-md)",
                padding: "var(--space-sm)",
                marginBottom: "var(--space-sm)",
              }}
            >
              {targets.map((item) => {
                const candidates = matches[item.id];
                const hasMatch = (candidates?.length ?? 0) > 0;
                const error = errors[item.id];
                return (
                  <div
                    key={item.id}
                    style={{
                      display: "grid",
                      gridTemplateColumns: "minmax(150px, 240px) minmax(0, 1fr)",
                      gap: "0.6rem",
                      alignItems: "center",
                    }}
                  >
                    <span className="text-sm" style={{ overflowWrap: "anywhere" }}>
                      {MEDIA_ICONS[item.media_type] ?? ""} {item.name}
                      {item.release_year ? ` (${item.release_year})` : ""}
                    </span>
                    {hasMatch ? (
                      <select
                        className="select"
                        style={{ width: "100%" }}
                        value={String(chosen[item.id] ?? 0)}
                        onChange={(e) => setChosen({ ...chosen, [item.id]: Number(e.target.value) })}
                        aria-label={item.name}
                      >
                        {candidates!.map((c, i) => (
                          <option key={c.external_id ?? `${i}-${c.title}`} value={String(i)}>
                            {candidateLabel(c)}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <span
                        className="text-sm"
                        style={{
                          color: error ? "var(--color-danger)" : "var(--color-text-muted)",
                        }}
                      >
                        {error ? `⚠️ ${error}` : t("library.noRatings")}
                      </span>
                    )}
                  </div>
                );
              })}
            </div>


            {savedCount > 0 && (
              <p className="text-sm" style={{ color: "var(--color-done)", margin: "0 0 0.25rem" }}>
                {t("library.bulkScoreSaved", { count: savedCount })}
              </p>
            )}
            {saveErrors.length > 0 && (
              <div style={{ marginBottom: "0.25rem" }}>
                <p className="text-sm" style={{ color: "var(--color-danger)", margin: "0 0 0.25rem" }}>
                  {t("library.bulkScoreSaveError")}
                </p>
                <ul
                  style={{
                    margin: 0,
                    paddingLeft: "1.1rem",
                    fontSize: "var(--font-size-xs)",
                    color: "var(--color-danger)",
                  }}
                >
                  {saveErrors.slice(0, 10).map((err, i) => (
                    <li key={i}>{err}</li>
                  ))}
                  {saveErrors.length > 10 && (
                    <li>{t("csv.andMore", { count: saveErrors.length - 10 })}</li>
                  )}
                </ul>
              </div>
            )}

            <div className="modal-actions">
              <button className="btn btn-ghost" disabled={busy} onClick={onClose}>
                {t("common.close")}
              </button>
              <button
                className="btn btn-primary"
                disabled={busy || saveCount === 0 || fullySaved}
                onClick={() => void handleSave()}
              >
                {saving ? t("common.saving") : t("library.bulkScoreSave", { count: saveCount })}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

