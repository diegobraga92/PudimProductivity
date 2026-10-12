import type { ReactNode } from "react";

interface FormActionsProps {
  children: ReactNode;
  /** Extra classes on the row, e.g. "mt-md" to separate it from the fields. */
  className?: string;
}

/** Action row (primary + secondary buttons) shared by the app's form modals. */
export default function FormActions({ children, className }: FormActionsProps) {
  return (
    <div className={`flex${className ? ` ${className}` : ""}`} style={{ gap: "0.5rem" }}>
      {children}
    </div>
  );
}
