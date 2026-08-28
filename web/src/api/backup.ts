import config from "../config";
import { apiHeaders } from "./client";

export interface BackupImportResult {
  restored_at: string;
  row_counts: Record<string, number>;
}

async function handleError(response: Response, fallback: string): Promise<never> {
  const body = await response.json().catch(() => null);
  throw new Error(body?.error || fallback);
}

/**
 * Exports a full backup of the non-sensitive data as a downloadable JSON file.
 * The backend sends it with Content-Disposition: attachment.
 */
export async function exportBackup(): Promise<{ blob: Blob; filename: string }> {
  const res = await fetch(`${config.apiBaseUrl}/backup/export`, {
    headers: apiHeaders(),
  });
  if (!res.ok) await handleError(res, `Failed to export backup: ${res.status}`);

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
  const res = await fetch(`${config.apiBaseUrl}/backup/import`, {
    method: "POST",
    headers: apiHeaders(),
    body: file,
  });
  if (!res.ok) await handleError(res, `Failed to import backup: ${res.status}`);
  return res.json() as Promise<BackupImportResult>;
}
