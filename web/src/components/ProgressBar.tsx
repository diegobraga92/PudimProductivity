interface ProgressBarProps {
  value: number; // 0-100
  variant?: "default" | "habit" | "todo";
  height?: number;
}

export default function ProgressBar({
  value,
  variant = "default",
}: ProgressBarProps) {
  const clamped = Math.max(0, Math.min(100, value));

  return (
    <div className="progress-bar">
      <div
        className={`progress-bar-fill ${variant}`}
        style={{ width: `${clamped}%` }}
      />
    </div>
  );
}
