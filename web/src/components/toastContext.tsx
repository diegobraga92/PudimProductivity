import { createContext, useContext } from "react";

export interface Toast {
  id: string;
  icon: string;
  title: string;
  body?: string;
  createdAt: string; // ISO timestamp
}

export interface ToastContextValue {
  toasts: Toast[];
  pushToast: (toast: Omit<Toast, "id" | "createdAt">) => void;
  dismissToast: (id: string) => void;
}

export const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return ctx;
}

