import { useState, useCallback, type ReactNode } from "react";
import { ConfirmContext } from "./useConfirm";
import ConfirmDialog, { type ConfirmDialogOptions } from "./ConfirmDialog";

type ConfirmFn = (options: ConfirmDialogOptions) => Promise<boolean>;

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