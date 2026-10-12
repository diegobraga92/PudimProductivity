import { useQuery } from "@tanstack/react-query";
import { getHealth, type HealthResponse } from "../api/client";
import { useI18n } from "../i18n";
import { HeartPulseIcon } from "./icons";

/** Backend + desktop status page (the "Health" entry in the More menu). */
export default function HealthPage() {
  const { t } = useI18n();
  const { data: healthData } = useQuery<HealthResponse>({
    queryKey: ["health"],
    queryFn: getHealth,
    refetchInterval: 30_000,
  });

  const desktopVersion = window.desktop?.versions.app;
  const isBackendOk = healthData?.status === "ok" && healthData?.db === "connected";

  return (
    <div className="animate-fade-in">
      <div className="flex-center" style={{ gap: "var(--space-sm)", marginBottom: "var(--space-lg)" }}>
        <h2 className="page-heading" style={{ marginBottom: 0 }}>
          <HeartPulseIcon size={24} />
          {t("nav.health")}
        </h2>
        <span className={`badge ${isBackendOk ? "badge-done" : "badge-habit"}`}>
          {isBackendOk ? t("status.healthy") : t("status.issues")}
        </span>
      </div>

      {!healthData && !desktopVersion && (
        <div className="card" style={{ textAlign: "center", padding: "var(--space-xl)" }}>
          <p style={{ color: "var(--color-text-secondary)" }}>{t("status.checking")}</p>
        </div>
      )}

      {(healthData || desktopVersion) && (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
            gap: "var(--space-md)",
          }}
        >
          {healthData && (
            <>
              <div className="stat-card">
                <div
                  className="stat-card-value"
                  style={{
                    color: healthData.status === "ok" ? "var(--color-done)" : "var(--color-warning)",
                  }}
                >
                  {healthData.status}
                </div>
                <div className="stat-card-label">{t("nav.status")}</div>
              </div>
              <div className="stat-card">
                <div className="stat-card-value" style={{ fontSize: "var(--font-size-lg)" }}>
                  v{healthData.version}
                </div>
                <div className="stat-card-label">{t("status.backendVersion")}</div>
              </div>
              <div className="stat-card">
                <div
                  className="stat-card-value"
                  style={{
                    color: healthData.db === "connected" ? "var(--color-done)" : "var(--color-danger)",
                    fontSize: "var(--font-size-sm)",
                  }}
                >
                  {healthData.db}
                </div>
                <div className="stat-card-label">{t("status.database")}</div>
              </div>
            </>
          )}
          {desktopVersion && (
            <div className="stat-card">
              <div className="stat-card-value" style={{ fontSize: "var(--font-size-lg)" }}>
                v{desktopVersion}
              </div>
              <div className="stat-card-label">{t("status.desktopVersion")}</div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
