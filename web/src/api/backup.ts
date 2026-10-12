import config from "../config";
import { apiError, apiHeaders, request } from "./client";

export interface BackupImportResult {
  restored_at: string;
  row_counts: Record<string, number>;
}

/**
 * Exports a full backup of the non-sensitive data as a downloadable JSON file.
 * The backend sends it with Content-Disposition: attachment.
 */
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

/**
 * Restores a backup, replacing the current contents of every backed-up table.
 * The restore is transactional on the server, a malformed backup changes
 * nothing.
 */
export async function importBackup(file: File): Promise<BackupImportResult> {
  return request<BackupImportResult>("/backup/import", {
    method: "POST",
    body: file,
    action: "import backup",
  });
}

