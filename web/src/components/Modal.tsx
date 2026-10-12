import { useEffect, type ReactNode } from "react";

interface ModalProps {
  onClose: () => void;
  maxWidth?: number;
  /** Accessible name of the dialog, read out by screen readers on open. */
  ariaLabel?: string;
  children: ReactNode;
}

/** Modal shell on .modal-backdrop/.modal-dialog; Escape/backdrop closes and body scroll is locked. */
export default function Modal({ onClose, maxWidth = 440, ariaLabel, children }: ModalProps) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "";
    };
  }, [onClose]);

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div
        className="modal-dialog"
        style={{ maxWidth, maxHeight: "90vh", overflowY: "auto" }}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-label={ariaLabel}
      >
        {children}
      </div>
    </div>
  );
}
