interface StreakBadgeProps {
  current: number;
  longest: number;
}

import { useI18n } from "../i18n";

export default function StreakBadge({ current, longest }: StreakBadgeProps) {
  const { t } = useI18n();
  if (current === 0 && longest === 0) return null;

  return (
    <span className="streak-badge" title={t("streak.longest", { count: longest })}>
      <span className="streak-badge-fire">🔥</span>
      {current}
      {longest > current && (
        <span style={{ opacity: 0.6, fontSize: "0.65rem", marginLeft: "0.1rem" }}>
          {t("streak.best", { count: longest })}
        </span>
      )}
    </span>
  );
}
