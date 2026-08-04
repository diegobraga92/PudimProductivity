import Checkbox from "./Checkbox";

interface TaskCardProps {
  variant: "todo" | "list";
  title: string;
  done: boolean;
  onToggle: () => void;
  onDelete: () => void;
  onTitleClick?: () => void;
  animationDelay?: string;
  compact?: boolean;
}

export default function TaskCard({
  variant,
  title,
  done,
  onToggle,
  onDelete,
  onTitleClick,
  animationDelay,
  compact = false,
}: TaskCardProps) {
  const cardClass = variant === "todo" ? "card-todo" : "card-list";

  return (
    <div
      className={`card ${cardClass} ${done ? "card-done" : ""}`}
      style={{
        display: "flex",
        alignItems: "center",
        gap: "var(--space-sm)",
        padding: compact ? "0.6rem 0.85rem" : "0.75rem 1rem",
        animationDelay,
      }}
    >
      <Checkbox checked={done} onChange={onToggle} />
      <span
        style={{
          flex: 1,
          cursor: onTitleClick ? "pointer" : "default",
          textDecoration: done ? "line-through" : "none",
          color: done ? "var(--color-text-muted)" : "var(--color-text)",
          fontSize: compact ? "var(--font-size-sm)" : "var(--font-size-base)",
          fontWeight: 500,
          transition: "all var(--transition-fast)",
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
        }}
        onClick={onTitleClick}
      >
        {title}
      </span>
      <button
        className="btn btn-danger btn-sm"
        onClick={onDelete}
        aria-label={`Delete ${title}`}
      >
        ✕
      </button>
    </div>
  );
}