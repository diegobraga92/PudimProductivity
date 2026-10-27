import { useCallback, useEffect, useRef, useState } from "react";
import { landingIndex, moveItem } from "../utils/reorder";

/** Vertical travel before a press on the handle turns into a drag, in px. */
const DRAG_THRESHOLD_PX = 4;

const ARIA_KEYSHORTCUTS = "ArrowUp ArrowDown Home End";

export interface DragHandleProps {
  ref: (element: HTMLButtonElement | null) => void;
  onPointerDown: (event: React.PointerEvent<HTMLButtonElement>) => void;
  onKeyDown: (event: React.KeyboardEvent<HTMLButtonElement>) => void;
  "aria-keyshortcuts": string;
}

interface UseDragReorderOptions {
  /** Item ids in the order they are rendered. */
  ids: readonly string[];
  /** Called with the full new order and the id the user moved. */
  onReorder: (orderedIds: string[], movedId: string) => void;
  /** Called once a press turns into a real drag, e.g. to switch to manual sort. */
  onDragStart?: () => void;
}

interface UseDragReorderResult {
  /** Id currently being dragged, null when idle. */
  draggingId: string | null;
  /** Index the dragged item would land on, null when idle. */
  landingIndex: number | null;
  /** Ref for the row element of an item, used to measure its midpoint. */
  registerItem: (id: string) => (element: HTMLElement | null) => void;
  /** Props to spread on the drag handle button. */
  handleProps: (id: string) => DragHandleProps;
}

/**
 * Pointer and keyboard reordering for a vertical list. Rows are reordered once
 * on drop (or on arrow keys) rather than live, so the caller only has to apply
 * and persist a final array.
 */
export function useDragReorder({
  ids,
  onReorder,
  onDragStart,
}: UseDragReorderOptions): UseDragReorderResult {
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [landing, setLanding] = useState<number | null>(null);

  const idsRef = useRef(ids);

  const landingRef = useRef<number | null>(null);
  const rows = useRef(new Map<string, HTMLElement>());
  const handles = useRef(new Map<string, HTMLButtonElement>());
  const rowRefs = useRef(new Map<string, (element: HTMLElement | null) => void>());
  const handleRefs = useRef(new Map<string, (element: HTMLButtonElement | null) => void>());
  const detach = useRef<(() => void) | null>(null);

  // Event handlers and pointer listeners read the ids through the ref, so they
  // always see the latest order without being re-created on every render.
  useEffect(() => {
    idsRef.current = ids;
  }, [ids]);

  useEffect(() => () => detach.current?.(), []);

  const updateLanding = (value: number | null) => {
    landingRef.current = value;
    setLanding(value);
  };

  const registerItem = useCallback((id: string) => {
    let callback = rowRefs.current.get(id);
    if (!callback) {
      callback = (element: HTMLElement | null) => {
        if (element) rows.current.set(id, element);
        else rows.current.delete(id);
      };
      rowRefs.current.set(id, callback);
    }
    return callback;
  }, []);

  const handleProps = useCallback(
    (id: string): DragHandleProps => {
      let ref = handleRefs.current.get(id);
      if (!ref) {
        ref = (element: HTMLButtonElement | null) => {
          if (element) handles.current.set(id, element);
          else handles.current.delete(id);
        };
        handleRefs.current.set(id, ref);
      }

      const startDrag = (event: React.PointerEvent<HTMLButtonElement>) => {
        if (event.button !== 0) return;

        const handle = event.currentTarget;
        const startY = event.clientY;
        const from = idsRef.current.indexOf(id);
        if (from === -1) return;

        handle.setPointerCapture?.(event.pointerId);
        let active = false;

        const midpointsOfOthers = () =>
          idsRef.current
            .filter((other) => other !== id)
            .map((other) => {
              const row = rows.current.get(other);
              if (!row) return Number.POSITIVE_INFINITY;
              const rect = row.getBoundingClientRect();
              return rect.top + rect.height / 2;
            });

        const onPointerMove = (moveEvent: PointerEvent) => {
          if (!active) {
            if (Math.abs(moveEvent.clientY - startY) < DRAG_THRESHOLD_PX) return;
            active = true;
            setDraggingId(id);
            document.body.classList.add("reorder-active");
            onDragStart?.();
          }
          updateLanding(landingIndex(midpointsOfOthers(), moveEvent.clientY));
        };

        const stop = (commit: boolean) => {
          detach.current?.();
          detach.current = null;
          handle.releasePointerCapture?.(event.pointerId);
          document.body.classList.remove("reorder-active");

          const target = landingRef.current;
          const wasDragging = active;
          active = false;
          setDraggingId(null);
          updateLanding(null);

          if (!commit || !wasDragging || target === null) return;

          const current = idsRef.current;
          onReorder(moveItem(current, current.indexOf(id), target), id);
        };

        const onPointerUp = () => stop(true);
        const onPointerCancel = () => stop(false);
        const onEscape = (keyEvent: KeyboardEvent) => {
          if (keyEvent.key === "Escape") stop(false);
        };

        window.addEventListener("pointermove", onPointerMove);
        window.addEventListener("pointerup", onPointerUp);
        window.addEventListener("pointercancel", onPointerCancel);
        window.addEventListener("keydown", onEscape);

        detach.current = () => {
          window.removeEventListener("pointermove", onPointerMove);
          window.removeEventListener("pointerup", onPointerUp);
          window.removeEventListener("pointercancel", onPointerCancel);
          window.removeEventListener("keydown", onEscape);
        };
      };

      const onKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
        const current = idsRef.current;
        const from = current.indexOf(id);
        if (from === -1) return;

        let to: number;
        switch (event.key) {
          case "ArrowUp":
            to = Math.max(0, from - 1);
            break;
          case "ArrowDown":
            to = Math.min(current.length - 1, from + 1);
            break;
          case "Home":
            to = 0;
            break;
          case "End":
            to = current.length - 1;
            break;
          default:
            return;
        }

        event.preventDefault();
        if (to === from) return;

        onReorder(moveItem(current, from, to), id);
        // The row is re-inserted at its new key position, which drops focus, so
        // put it back on the same handle.
        window.requestAnimationFrame(() => handles.current.get(id)?.focus());
      };

      return {
        ref,
        onPointerDown: startDrag,
        onKeyDown,
        "aria-keyshortcuts": ARIA_KEYSHORTCUTS,
      };
    },
    [onDragStart, onReorder]
  );

  return { draggingId, landingIndex: landing, registerItem, handleProps };
}
