import type { ScoreProvidersConfig } from "../api/admin";
import type { MediaType } from "../api/library";
import { useI18n } from "../i18n";
import { MEDIA_ICONS, MEDIA_LABEL_KEYS, MEDIA_TYPES } from "../utils/media";

interface ScoreProviderTableProps {
  providers: ScoreProvidersConfig["providers"];
  /** Selected provider name per media type. */
  assignments: Record<string, string>;
  onAssign: (mediaType: MediaType, provider: string) => void;
}

/** Media type → rating provider mapping table (one row per media type). */
export default function ScoreProviderTable({
  providers,
  assignments,
  onAssign,
}: ScoreProviderTableProps) {
  const { t } = useI18n();

  return (
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
                onChange={(e) => onAssign(mt, e.target.value)}
              >
                <option value="">{t("serverSettings.providerNone")}</option>
                {providers
                  .filter((p) => p.supported_types.includes(mt))
                  .map((p) => (
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
  );
}
