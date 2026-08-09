import { useEffect } from "react";
import config from "../config";

interface ErrorReport {
  message: string;
  stack?: string;
  context?: string;
}

/**
 * Reports client-side errors (window.onerror + unhandledrejection) to the
 * backend POST /api/v1/errors beacon, where they are logged with trace context.
 * Debounced to at most one request per second to avoid flooding.
 */
export function useErrorReporter(): void {
  useEffect(() => {
    let lastSent = 0;

    const report = (report: ErrorReport) => {
      const now = Date.now();
      if (now - lastSent < 1_000) return;
      lastSent = now;

      fetch(`${config.apiBaseUrl}/errors`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Error-Source": "web",
        },
        body: JSON.stringify(report),
        // Beacon-style: don't block the page on this.
        keepalive: true,
      }).catch(() => {
        /* best-effort: never let error reporting break the app */
      });
    };

    const onError = (event: ErrorEvent) => {
      report({
        message: event.message || "Unknown error",
        stack: event.error?.stack,
        context: `at ${event.filename || ""}:${event.lineno}:${event.colno}`,
      });
    };

    const onRejection = (event: PromiseRejectionEvent) => {
      const reason = event.reason as { message?: string; stack?: string } | undefined;
      report({
        message: reason?.message || "Unhandled promise rejection",
        stack: reason?.stack,
      });
    };

    window.addEventListener("error", onError);
    window.addEventListener("unhandledrejection", onRejection);

    return () => {
      window.removeEventListener("error", onError);
      window.removeEventListener("unhandledrejection", onRejection);
    };
  }, []);
}
