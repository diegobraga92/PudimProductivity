import config from "../config";
import { apiError, apiHeaders, request } from "./client";

export interface BackupImportResult {
  restored_at: string;
  row_counts: Record<string, number>;
}

/** Download a JSON backup of the non-sensitive data. */
export async function exportBackup(): Promise<{ blob: Blob; filename: string }> {
  const res = await fetch(`${config.apiBaseUrl}/backup/export`, {
    headers: apiHeaders(),
  });
  if (!res.ok) throw await apiError(res, `Failed to export backup: ${res.status}`);

  const blob = await res.blob();
  const disposition = res.headers.get("Content-Disposition") ?? "";
  const match = /filename="?([^"]+)"?/.exec(disposition);
  const fallback = `pudim-backup-${new Date().toISOString().slice(0, 10)}.json`;
  return { blob, filename: match?.[1] ?? fallback };
}

/** Restore a backup, replacing every backed-up table. Server-side transactional: malformed input changes nothing. */
export async function importBackup(file: File): Promise<BackupImportResult> {
  return request<BackupImportResult>("/backup/import", {
    method: "POST",
    body: file,
    action: "import backup",
  });
}

