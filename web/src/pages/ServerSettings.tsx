import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import {
  getScoreProviders,
  saveScoreProviders,
  type ScoreProvidersConfig,
  type ScoreProvidersUpdate,
} from "../api/admin";
import { getDevRole, setDevRole } from "../api/client";
import { searchLibraryScores } from "../api/library";
import { useI18n } from "../i18n";

const MEDIA_TYPES = ["movie", "series", "game", "book"] as const;
const MEDIA_LABEL_KEYS: Record<(typeof MEDIA_TYPES)[number], string> = {
  movie: "library.movie",
  series: "library.series",
  game: "library.game",
  book: "library.book",
};
const MEDIA_ICONS: Record<(typeof MEDIA_TYPES)[number], string> = {
  movie: "🎬",
  series: "📺",
  game: "🎮",
  book: "📚",
};

// Labels for the extra per-provider settings fields returned by the backend
// (score_providers.settings JSONB). Keys without an entry fall back to the raw
// field name.
const SETTING_LABEL_KEYS: Record<string, string> = {
  client_secret: "serverSettings.clientSecret",
};

/**
 * Server Settings (admin): manages the library score-provider configuration at
 * runtime — which rating provider serves each media type, plus per-provider API
 * keys. Replaces the .env-only configuration (SCORE_PROVIDER_* / *_API_KEY,
 * which now only act as a one-time bootstrap). Admin-only on the backend; this
 * page offers a dev admin-mode toggle consistent with the dev identity headers.
 */
export default function ServerSettings() {
  const queryClient = useQueryClient();
  const { t } = useI18n();

  const [role, setRole] = useState<string>(getDevRole());
  const isAdmin = role === "admin";

  const [assignments, setAssignments] = useState<Record<string, string>>({ movie: "", series: "", game: "", book: "" });
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({});
  const [baseUrls, setBaseUrls] = useState<Record<string, string>>({});
  // Per-provider extra settings (e.g. IGDB's client_secret), keyed by provider
  // name then setting key. Only non-empty entries are sent — empty = keep.
  const [settings, setSettings] = useState<Record<string, Record<string, string>>>({});
  const [lookupEnabled, setLookupEnabled] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [savedFlash, setSavedFlash] = useState(false);

  const [testTitle, setTestTitle] = useState("The Legend of Zelda: Tears of the Kingdom");
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState<string | null>(null);

  const { data, error, isLoading } = useQuery({
    queryKey: ["admin", "score-providers", role],
    queryFn: getScoreProviders,
    enabled: isAdmin,
    retry: false,
  });

  // Populate the form whenever fresh config arrives.
  useEffect(() => {
    if (!data) return;
    setAssignments({
      movie: data.movie_provider,
      series: data.series_provider,
      game: data.game_provider,
      book: data.book_provider,
    });
    setLookupEnabled(data.lookup_enabled);
    setDirty(false);
  }, [data]);

  const save = useMutation({
    mutationFn: async (): Promise<ScoreProvidersConfig> => {
      const providers = (data?.providers ?? []).map((p) => {
        // Same "empty = keep" rule as the API key: only send a setting when the
        // admin typed a value. Omitting the key leaves the stored secret alone.
        const settingsPayload: Record<string, string> = {};
        for (const k of p.settings_keys ?? []) {
          const v = settings[p.name]?.[k];
          if (v) settingsPayload[k] = v;
        }
        return {
          name: p.name,
          // Empty input = keep the stored key; only send a new key when typed.
          api_key: apiKeys[p.name] ? apiKeys[p.name] : null,
          base_url: null,
          ...(Object.keys(settingsPayload).length > 0 ? { settings: settingsPayload } : {}),
        };
      });
      const req: ScoreProvidersUpdate = {
        movie_provider: assignments.movie,
        series_provider: assignments.series,
        game_provider: assignments.game,
        book_provider: assignments.book,
        lookup_enabled: lookupEnabled,
        providers,
      };
      return saveScoreProviders(req);
    },
    onSuccess: (res) => {
      queryClient.setQueryData(["admin", "score-providers", role], res);
      setApiKeys({});
      setBaseUrls({});
      setSettings({});
      setDirty(false);
      setSavedFlash(true);
      window.setTimeout(() => setSavedFlash(false), 3000);
    },
  });

  function toggleAdmin() {
    const next = isAdmin ? "user" : "admin";
    setDevRole(next as "admin" | "user");
    setRole(next);
    setTestResult(null);
    queryClient.invalidateQueries({ queryKey: ["admin"] });
  }

  async function testLookup() {
    setTesting(true);
    setTestResult(null);
    try {
      const hits = await searchLibraryScores(testTitle.trim() || "Mario", "game");
      setTestResult(
        hits.length > 0
          ? t("serverSettings.testResult", { title: hits[0].title, count: hits.length })
          : t("serverSettings.testNoResult"),
      );
    } catch (err) {
      setTestResult(err instanceof Error ? err.message : String(err));
    } finally {
      setTesting(false);
    }
  }

  const providerOptions = (mediaType: (typeof MEDIA_TYPES)[number]) =>
    (data?.providers ?? []).filter((p) => p.supported_types.includes(mediaType));

  return (
    <div className="animate-fade-in">
      <h2 className="page-heading">{t("serverSettings.title")}</h2>

      {!isAdmin && (
        <div className="card" style={{ maxWidth: 640, padding: "var(--space-md)", marginBottom: "var(--space-md)" }}>
          <p style={{ fontWeight: 600, margin: "0 0 0.25rem" }}>{t("serverSettings.adminRequired")}</p>
          <p className="text-sm text-secondary" style={{ margin: "0 0 var(--space-sm)" }}>
            {t("serverSettings.adminRequiredDesc")}
          </p>
          <button className="btn" onClick={toggleAdmin}>
            {t("serverSettings.enableAdmin")}
          </button>
        </div>
      )}

      {isAdmin && (
        <div style={{ display: "grid", gap: "var(--space-md)", maxWidth: 720 }}>
          {isLoading && <p className="text-sm text-secondary">{t("common.loading")}</p>}
          {error && (
            <div className="card" style={{ padding: "var(--space-md)", borderColor: "var(--color-danger)" }}>
              <p style={{ margin: 0, color: "var(--color-danger)" }}>
                {error instanceof Error ? error.message : String(error)}
              </p>
            </div>
          )}

          {data && (
            <>
              <div className="card" style={{ padding: "var(--space-md)" }}>
                <div className="flex-center" style={{ justifyContent: "space-between", marginBottom: "0.4rem" }}>
                  <h3 style={{ margin: 0, fontSize: "var(--font-size-lg)" }}>{t("serverSettings.scoreLookup")}</h3>
                  <label className="flex-center" style={{ gap: "0.35rem" }}>
                    <input
                      type="checkbox"
                      checked={lookupEnabled}
                      onChange={(e) => { setLookupEnabled(e.target.checked); setDirty(true); }}
                    />
                    <span className="text-sm">{t("serverSettings.enableLookup")}</span>
                  </label>
                </div>
                <p className="text-sm text-secondary" style={{ margin: "0 0 var(--space-md)" }}>
                  {t("serverSettings.scoreLookupDesc")}
                </p>

                <table style={{ width: "100%", borderCollapse: "collapse", fontSize: "var(--font-size-sm)" }}>
                  <thead>
                    <tr>
                      <th style={{ textAlign: "left", padding: "0.25rem 0.5rem", borderBottom: "1px solid var(--color-border-light)" }}>
                        {t("serverSettings.mediaType")}
                      </th>
                      <th style={{ textAlign: "left", padding: "0.25rem 0.5rem", borderBottom: "1px solid var(--color-border-light)" }}>
                        {t("serverSettings.provider")}
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {MEDIA_TYPES.map((mt) => (
                      <tr key={mt}>
                        <td style={{ padding: "0.35rem 0.5rem" }}>
                          {MEDIA_ICONS[mt]} {t(MEDIA_LABEL_KEYS[mt])}
                        </td>
                        <td style={{ padding: "0.35rem 0.5rem" }}>
                          <select
                            className="select"
                            style={{ minWidth: 180 }}
                            value={assignments[mt]}
                            onChange={(e) => { setAssignments({ ...assignments, [mt]: e.target.value }); setDirty(true); }}
                          >
                            <option value="">{t("serverSettings.providerNone")}</option>
                            {providerOptions(mt).map((p) => (
                              <option key={p.name} value={p.name}>
                                {p.name}
                              </option>
                            ))}
                          </select>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>


              <div className="card" style={{ padding: "var(--space-md)" }}>
                <h3 style={{ margin: "0 0 var(--space-sm)", fontSize: "var(--font-size-lg)" }}>🔑 API keys</h3>
                {data.providers.map((p) => (
                  <div key={p.name} style={{ display: "grid", gap: "0.3rem", marginBottom: "var(--space-md)" }}>
                    <p className="text-sm" style={{ fontWeight: 600, margin: 0 }}>
                      {p.name}
                      {p.name === "igdb" && (
                        <span className="text-secondary" style={{ fontWeight: 400 }}> — {t("serverSettings.igdbHint")}</span>
                      )}
                    </p>
                    <input
                      className="input"
                      type="password"
                      placeholder={p.api_key_set ? t("serverSettings.apiKeyStored") : t("serverSettings.apiKey")}
                      value={apiKeys[p.name] ?? ""}
                      autoComplete="off"
                      onChange={(e) => { setApiKeys({ ...apiKeys, [p.name]: e.target.value }); setDirty(true); }}
                    />
                    {(p.settings_keys ?? []).map((k) => (
                      <input
                        key={k}
                        className="input"
                        type="password"
                        placeholder={`${t(SETTING_LABEL_KEYS[k] ?? "serverSettings.setting")}${p.settings_set?.[k] ? ` — ${t("serverSettings.settingStored")}` : ""}`}
                        value={settings[p.name]?.[k] ?? ""}
                        autoComplete="off"
                        onChange={(e) => {
                          setSettings({ ...settings, [p.name]: { ...(settings[p.name] ?? {}), [k]: e.target.value } });
                          setDirty(true);
                        }}
                      />
                    ))}
                    <input
                      className="input"
                      placeholder={t("serverSettings.baseUrl")}
                      value={baseUrls[p.name] ?? p.base_url}
                      onChange={(e) => { setBaseUrls({ ...baseUrls, [p.name]: e.target.value }); setDirty(true); }}
                    />
                  </div>
                ))}
              </div>

              <div className="card" style={{ padding: "var(--space-md)" }}>
                <h3 style={{ margin: "0 0 var(--space-sm)", fontSize: "var(--font-size-lg)" }}>🧪 {t("serverSettings.testLookup")}</h3>
                <div className="flex-center" style={{ gap: "0.4rem", flexWrap: "wrap" }}>
                  <input
                    className="input"
                    style={{ flex: 1, minWidth: 220 }}
                    value={testTitle}
                    onChange={(e) => setTestTitle(e.target.value)}
                  />
                  <button className="btn" disabled={testing || !lookupEnabled} onClick={testLookup}>
                    {testing ? t("serverSettings.testing") : t("serverSettings.testLookup")}
                  </button>
                </div>
                {!lookupEnabled && (
                  <p className="text-sm text-secondary" style={{ margin: "0.4rem 0 0" }}>
                    {t("serverSettings.lookupDisabledNote")}
                  </p>
                )}
                {testResult && (
                  <p
                    className="text-sm"
                    style={{ margin: "0.4rem 0 0", color: testResult.startsWith("Tested") ? undefined : "var(--color-danger)" }}
                  >
                    {testResult}
                  </p>
                )}
              </div>

              <div className="flex-center" style={{ gap: "0.5rem" }}>
                <button className="btn btn-primary" disabled={!dirty || save.isPending} onClick={() => save.mutate()}>
                  {save.isPending ? t("common.saving") : t("serverSettings.save")}
                </button>
                {save.error && (
                  <span className="text-sm" style={{ color: "var(--color-danger)" }}>
                    {save.error instanceof Error ? save.error.message : String(save.error)}
                  </span>
                )}
                {savedFlash && <span className="text-sm" style={{ color: "var(--color-done)" }}>{t("serverSettings.saved")}</span>}
                <button className="btn btn-ghost" onClick={() => { setDevRole("user"); setRole("user"); }}>
                  {t("serverSettings.disableAdmin")}
                </button>
                <span className="text-sm text-secondary" style={{ marginLeft: "auto" }}>
                  {t("serverSettings.adminMode")}: <strong>{isAdmin ? "admin" : "user"}</strong>
                </span>
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}

