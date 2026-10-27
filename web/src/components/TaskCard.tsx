import Checkbox from "./Checkbox";
import { useI18n } from "../i18n";
import type { DragHandleProps } from "../hooks/useDragReorder";

interface TaskCardProps {
  variant: "todo" | "list";
  title: string;
  done: boolean;
  onToggle: () => void;
  onDelete: () => void;
  onTitleClick?: () => void;
  animationDelay?: string;
  compact?: boolean;
  /** Ref for the row element, drag measurement reads its box. */
  rowRef?: (element: HTMLDivElement | null) => void;
  /** Props for the drag handle, omitted when the list cannot be reordered. */
  dragHandle?: DragHandleProps;
  /** True while this row is the one being dragged. */
  dragging?: boolean;
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
  rowRef,
  dragHandle,
  dragging = false,
}: TaskCardProps) {
  const { t } = useI18n();
  const cardClass = variant === "todo" ? "card-todo" : "card-list";

  return (
    <div
      ref={rowRef}
      className={`card ${cardClass} ${done ? "card-done" : ""} ${dragging ? "card-dragging" : ""}`}
      style={{
        display: "flex",
        alignItems: "center",
        gap: "var(--space-sm)",
        padding: compact ? "0.6rem 0.85rem" : "0.75rem 1rem",
        animationDelay,
      }}
    >
      {dragHandle && (
        <button
          type="button"
          className="reorder-handle"
          {...dragHandle}
          aria-label={t("tasks.reorderAria", { title })}
          title={t("tasks.reorderAria", { title })}
        >
          ⠿
        </button>
      )}
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
        className="btn btn-danger btn-sm task-delete"
        onClick={onDelete}
        aria-label={t("tasks.deleteAria", { title })}
      >
        ✕
      </button>
    </div>
  );
}