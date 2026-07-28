import { createContext, useContext } from "react";
import type { ConfirmDialogOptions } from "./ConfirmDialog";

interface ConfirmFn {
  (options: ConfirmDialogOptions): Promise<boolean>;
}

interface ConfirmContextValue {
  confirm: ConfirmFn;
}

export const ConfirmContext = createContext<ConfirmContextValue | null>(null);

export function useConfirm(): ConfirmFn {
  const ctx = useContext(ConfirmContext);
  if (!ctx) {
    throw new Error("useConfirm must be used within a ConfirmProvider");
  }
  return ctx.confirm;
}