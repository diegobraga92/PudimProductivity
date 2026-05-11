interface StreakBadgeProps {
  current: number;
  longest: number;
}

export default function StreakBadge({ current, longest }: StreakBadgeProps) {
  if (current === 0 && longest === 0) return null;

  return (
    <span className="streak-badge" title={`Longest streak: ${longest} days`}>
      <span className="streak-badge-fire">🔥</span>
      {current}
      {longest > current && (
        <span style={{ opacity: 0.6, fontSize: "0.65rem", marginLeft: "0.1rem" }}>
          (best: {longest})
        </span>
      )}
    </span>
  );
}
