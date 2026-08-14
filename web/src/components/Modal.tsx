import { useEffect, type ReactNode } from "react";

interface ModalProps {
  onClose: () => void;
  maxWidth?: number;
  children: ReactNode;
}

/**
 * Reusable modal shell: backdrop + centered dialog with the app's existing
 * .modal-backdrop/.modal-dialog styling. Closes on Escape or backdrop click,
 * and locks body scroll while open. Mount/unmount controls visibility.
 */
export default function Modal({ onClose, maxWidth = 440, children }: ModalProps) {
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
      >
        {children}
      </div>
    </div>
  );
}
