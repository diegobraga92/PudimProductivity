import { useEffect, useRef } from "react";
import { useI18n } from "../i18n";

export interface ConfirmDialogOptions {
  title: string;
  message?: string;
  confirmLabel?: string;
  confirmVariant?: "danger" | "primary";
  cancelLabel?: string;
}

interface ConfirmDialogProps {
  open: boolean;
  options: ConfirmDialogOptions;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmDialog({
  open,
  options,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const { t } = useI18n();
  const confirmRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (open) {
      confirmRef.current?.focus();
    }
  }, [open]);

  useEffect(() => {
    if (!open) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onCancel();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, onCancel]);

  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  if (!open) return null;

  const confirmClassName =
    options.confirmVariant === "danger" ? "btn btn-danger" : "btn btn-primary";

  return (
    <div className="modal-backdrop" onClick={onCancel}>
      <div
        className="modal-dialog animate-fade-in"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
      >
        <h3 id="confirm-title" className="modal-title">
          {options.title}
        </h3>

        {options.message && (
          <p className="modal-message">{options.message}</p>
        )}

        <div className="modal-actions">
          <button className="btn btn-ghost" onClick={onCancel}>
            {options.cancelLabel ?? t("common.cancel")}
          </button>
          <button
            ref={confirmRef}
            className={confirmClassName}
            onClick={onConfirm}
          >
            {options.confirmLabel ?? t("common.confirm")}
          </button>
        </div>
      </div>
    </div>
  );
}