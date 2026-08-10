import { useQuery } from "@tanstack/react-query";
import { getWeeklyInsights, type InsightReport } from "../api/insights";

/** Formats a plain-text report as paragraphs for display. */
function paragraphs(text: string): string[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
}

/**
 * Phase 9a: weekly productivity report — completions, focus minutes, top
 * habits and new recipes, plus an optional LLM summary.
 */
export default function Insights() {
  const { data: report, isLoading, error } = useQuery<InsightReport>({
    queryKey: ["insights", "weekly"],
    queryFn: () => getWeeklyInsights(),
  });

  return (
    <div className="animate-fade-in">
      <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "var(--space-md)" }}>
        <h2 style={{ fontSize: "var(--font-size-xl)", fontWeight: 700 }}>🧠 Insights</h2>
        {report && (
          <span className="badge badge-habit">Week of {report.week_start}</span>
        )}
      </div>

      {isLoading && <p style={{ color: "var(--color-text-secondary)" }}>Crunching the numbers…</p>}
      {error && <p style={{ color: "var(--color-danger)" }}>{(error as Error).message}</p>}

      {report && (
        <>
          {/* Stats cards */}
          <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))", gap: "var(--space-sm)", marginBottom: "var(--space-lg)" }}>
            <div className="card stat-card">
              <div className="stat-card-value">{report.stats.total_completions}</div>
              <div className="stat-card-label">Completed · {report.stats.completions_per_day.toFixed(1)}/day</div>
            </div>
            <div className="card stat-card">
              <div className="stat-card-value">{report.stats.focus_minutes}m</div>
              <div className="stat-card-label">Focus · {report.stats.focus_sessions} session{report.stats.focus_sessions === 1 ? "" : "s"}</div>
            </div>
            <div className="card stat-card">
              <div className="stat-card-value">{report.stats.recipes_created}</div>
              <div className="stat-card-label">Recipes added</div>
            </div>
            <div className="card stat-card">
              <div className="stat-card-value">{(report.stats.top_habits ?? []).length}</div>
              <div className="stat-card-label">Top habits</div>
              <div style={{ fontSize: "var(--font-size-xs)", color: "var(--color-text-secondary)" }}>
                {(report.stats.top_habits ?? []).map((h) => h.title).join(", ") || "—"}
              </div>
            </div>
          </div>

          {/* LLM summary (only when the optional flag produced one) */}
          {report.llm_summary && (
            <div className="card" style={{ marginBottom: "var(--space-md)", background: "var(--color-primary-light)" }}>
              <div style={{ fontWeight: 600, marginBottom: "0.25rem" }}>✨ AI Coach</div>
              <p style={{ margin: 0, fontSize: "var(--font-size-sm)" }}>{report.llm_summary}</p>
            </div>
          )}

          {/* Template report */}
          <div className="card">
            <div style={{ fontWeight: 600, marginBottom: "0.5rem" }}>📈 Weekly report</div>
            {paragraphs(report.report_text).map((line, i) => (
              <p key={i} style={{ margin: "0.25rem 0", fontSize: "var(--font-size-sm)" }}>
                {line}
              </p>
            ))}
          </div>
        </>
      )}

      {!isLoading && !error && !report && (
        <div className="empty-state">
          <div className="empty-state-icon">🧠</div>
          <p className="empty-state-text">No report available yet.</p>
        </div>
      )}
    </div>
  );
}
