import { createContext, useContext, useState, useCallback, type ReactNode } from "react";
import ConfirmDialog, { type ConfirmDialogOptions } from "./ConfirmDialog";

interface ConfirmFn {
  (options: ConfirmDialogOptions): Promise<boolean>;
}

interface ConfirmContextValue {
  confirm: ConfirmFn;
}

const ConfirmContext = createContext<ConfirmContextValue | null>(null);

export function useConfirm(): ConfirmFn {
  const ctx = useContext(ConfirmContext);
  if (!ctx) {
    throw new Error("useConfirm must be used within a ConfirmProvider");
  }
  return ctx.confirm;
}

interface PendingConfirm {
  resolve: (value: boolean) => void;
  options: ConfirmDialogOptions;
}

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [pending, setPending] = useState<PendingConfirm | null>(null);

  const confirm = useCallback<ConfirmFn>((options) => {
    return new Promise<boolean>((resolve) => {
      const entry: PendingConfirm = { resolve, options };
      setPending(entry);
    });
  }, []);

  const handleConfirm = useCallback(() => {
    if (pending) {
      pending.resolve(true);
      setPending(null);
    }
  }, [pending]);

  const handleCancel = useCallback(() => {
    if (pending) {
      pending.resolve(false);
      setPending(null);
    }
  }, [pending]);

  return (
    <ConfirmContext.Provider value={{ confirm }}>
      {children}
      <ConfirmDialog
        open={!!pending}
        options={pending?.options ?? { title: "" }}
        onConfirm={handleConfirm}
        onCancel={handleCancel}
      />
    </ConfirmContext.Provider>
  );
}